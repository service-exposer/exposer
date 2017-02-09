package exposer

import (
	"encoding/json"
	"io"
	"net"
	"sync"

	"github.com/inconshreveable/muxado"
)

type Protocal struct {
	conn             net.Conn
	isHandshakeDone  bool
	handshakeDecoder *json.Decoder

	// handle handshake
	On func(proto *Protocal, cmd string, details []byte) error
}

func NewProtocal(conn net.Conn) *Protocal {
	return &Protocal{
		conn:             conn,
		isHandshakeDone:  false,
		handshakeDecoder: json.NewDecoder(conn),
	}
}

func (proto *Protocal) Reply(cmd string, details interface{}) error {
	if proto.isHandshakeDone {
		panic("protoport handshake is done, unexpect Reply call")
	}

	return json.NewEncoder(proto.conn).Encode(&HandshakeOutgoing{
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

func (proto *Protocal) Multiplex(isClient bool) muxado.Session {
	proto.isHandshakeDone = true

	if isClient {
		return muxado.Client(newReadWriteCloser(proto.handshakeDecoder.Buffered(), proto.conn), nil)
	}

	return muxado.Server(newReadWriteCloser(proto.handshakeDecoder.Buffered(), proto.conn), nil)
}

func (proto *Protocal) Forward(conn net.Conn) {
	defer proto.conn.Close()
	defer conn.Close()

	proto.isHandshakeDone = true

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(conn, io.MultiReader(proto.handshakeDecoder.Buffered(), proto.conn))
	}()

	go func() {
		defer wg.Done()
		io.Copy(proto.conn, conn)
	}()
	wg.Done()
}

func (proto *Protocal) Request(cmd string, details interface{}) {
	err := proto.Reply(cmd, details)
	if err != nil {
		proto.conn.Close()
		return
	}

	proto.Handle()
}

func (proto *Protocal) Handle() {
	defer proto.conn.Close()

	if proto.On == nil {
		panic("not set Protocal.On")
	}

	var handshake HandshakeIncoming
	for {
		err := proto.handshakeDecoder.Decode(&handshake)
		if err != nil {
			// TODO: handle error
			return
		}

		err = proto.On(proto, handshake.Command, handshake.Details)
		if err != nil {
			// TODO: handle error
			return
		}
	}
}
