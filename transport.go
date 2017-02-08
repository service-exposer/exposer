package exposer

import (
	"encoding/json"
	"io"
	"net"
	"sync"

	"github.com/inconshreveable/muxado"
)

type Transport struct {
	conn             net.Conn
	isHandshakeDone  bool
	handshakeDecoder *json.Decoder

	// handle handshake
	On func(trans *Transport, cmd string, details []byte) error
}

func NewTransport(conn net.Conn) *Transport {
	return &Transport{
		conn:             conn,
		isHandshakeDone:  false,
		handshakeDecoder: json.NewDecoder(conn),
	}
}

func (trans *Transport) Reply(cmd string, details interface{}) error {
	if trans.isHandshakeDone {
		panic("transport handshake is done, unexpect Reply call")
	}

	return json.NewEncoder(trans.conn).Encode(&HandshakeOutgoing{
		Command: cmd,
		Details: details,
	})
}

func newReadWriteCloser(buffered io.Reader, conn net.Conn) io.ReadWriteCloser {
	type readWriteCloser struct {
		io.Reader
		io.Writer
		io.Closer
	}

	return &readWriteCloser{
		Reader: io.MultiReader(buffered, conn),
		Writer: conn,
		Closer: conn,
	}
}

func (trans *Transport) Multiplex(isClient bool) muxado.Session {
	trans.isHandshakeDone = true

	if isClient {
		return muxado.Client(newReadWriteCloser(trans.handshakeDecoder.Buffered(), trans.conn), nil)
	}

	return muxado.Server(newReadWriteCloser(trans.handshakeDecoder.Buffered(), trans.conn), nil)
}

func (trans *Transport) Forward(conn net.Conn) {
	defer trans.conn.Close()
	defer conn.Close()

	trans.isHandshakeDone = true

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(conn, io.MultiReader(trans.handshakeDecoder.Buffered(), trans.conn))
	}()

	go func() {
		defer wg.Done()
		io.Copy(trans.conn, conn)
	}()
	wg.Done()
}

func (trans *Transport) Request(cmd string, details interface{}) {
	err := trans.Reply(cmd, details)
	if err != nil {
		trans.conn.Close()
		return
	}

	trans.Handle()
}

func (trans *Transport) Handle() {
	defer trans.conn.Close()

	if trans.On == nil {
		panic("not set Transport.On")
	}

	var handshake HandshakeIncoming
	for {
		err := trans.handshakeDecoder.Decode(&handshake)
		if err != nil {
			// TODO: handle error
			return
		}

		err = trans.On(trans, handshake.Command, handshake.Details)
		if err != nil {
			// TODO: handle error
			return
		}
	}
}
