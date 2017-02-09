package exposer

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/inconshreveable/muxado"
)

type HandshakeHandleFunc func(proto *Protocal, cmd string, details []byte) error
type Protocal struct {
	conn             net.Conn
	isHandshakeDone  bool
	handshakeDecoder *json.Decoder

	// handle handshake
	On HandshakeHandleFunc
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

	data, err := json.Marshal(&HandshakeOutgoing{
		Command: cmd,
		Details: details,
	})
	if err != nil {
		return err
	}

	_, err = proto.conn.Write(data)
	return err
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
		defer conn.Close()
		io.Copy(conn, io.TeeReader(io.MultiReader(proto.handshakeDecoder.Buffered(), proto.conn), os.Stdout))
	}()

	go func() {
		defer wg.Done()
		defer proto.conn.Close()
		io.Copy(proto.conn, io.TeeReader(conn, os.Stdout))
	}()
	wg.Wait()
}

func (proto *Protocal) Request(cmd string, details interface{}) {
	err := proto.Reply(cmd, details)
	if err != nil {
		log.Print(".Request ", cmd, " ", err)
		proto.conn.Close()
		return
	}

	log.Println("start .Handle ", cmd)
	proto.Handle()
}

func (proto *Protocal) Handle() {
	defer proto.conn.Close()

	if proto.On == nil {
		panic("not set Protocal.On")
	}

	var handshake HandshakeIncoming
	for !proto.isHandshakeDone {
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
