package exposer

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/inconshreveable/muxado"
)

type HandshakeHandleFunc func(proto *Protocal, cmd string, details []byte) error
type Protocal struct {
	conn             net.Conn
	isHandshakeDone  bool
	handshakeDecoder *json.Decoder
	eventbus         chan HandshakeIncoming
	done             chan struct{}

	setErrOnce *sync.Once
	err        error

	// handle handshake
	mutex_On *sync.Mutex
	On       HandshakeHandleFunc
}

func NewProtocal(conn net.Conn) *Protocal {
	return &Protocal{
		conn:             conn,
		isHandshakeDone:  false,
		handshakeDecoder: json.NewDecoder(conn),
		eventbus:         make(chan HandshakeIncoming),
		done:             make(chan struct{}),
		setErrOnce:       new(sync.Once),
		err:              nil,
		mutex_On:         new(sync.Mutex),
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

var (
	muxadoMutex = new(sync.Mutex)
)

func (proto *Protocal) Multiplex(isClient bool) muxado.Session {
	proto.isHandshakeDone = true

	muxadoMutex.Lock()
	defer muxadoMutex.Unlock()

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
		io.Copy(conn, io.MultiReader(proto.handshakeDecoder.Buffered(), proto.conn))
	}()

	go func() {
		defer wg.Done()
		defer proto.conn.Close()
		io.Copy(proto.conn, conn)
	}()
	wg.Wait()
}

func (proto *Protocal) Request(cmd string, details interface{}) {
	err := proto.Reply(cmd, details)
	if err != nil {
		proto.conn.Close()
		return
	}

	proto.Handle()
}

func (proto *Protocal) Emit(event string, details interface{}) (err error) {
	var data []byte
	data, err = json.Marshal(&HandshakeOutgoing{
		Command: event,
		Details: details,
	})
	if err != nil {
		return err
	}

	var handshake HandshakeIncoming
	err = json.Unmarshal(data, &handshake)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			err = errors.New("conn closed")
		}
	}()
	proto.eventbus <- handshake
	return nil
}

func (proto *Protocal) Handle() {
	defer close(proto.eventbus)
	defer proto.conn.Close()

	if proto.On == nil {
		panic("not set Protocal.On")
	}

	go func() {
		defer proto.conn.Close()

		for handshake := range proto.eventbus {
			err := func() error {
				proto.mutex_On.Lock()
				defer proto.mutex_On.Unlock()
				return proto.On(proto, handshake.Command, handshake.Details)
			}()
			if err != nil {
				proto.Shutdown(err)
				return
			}
		}
	}()

	var handshake HandshakeIncoming
	for !proto.isHandshakeDone {
		err := proto.handshakeDecoder.Decode(&handshake)

		if err != nil {
			proto.Shutdown(err)
			return
		}
		err = func() error {
			proto.mutex_On.Lock()
			defer proto.mutex_On.Unlock()
			return proto.On(proto, handshake.Command, handshake.Details)
		}()
		if err != nil {
			proto.Shutdown(err)
			return
		}

	}

}

func (proto *Protocal) Shutdown(err error) {
	if err == nil {
		panic("err cannot be nil")
	}
	proto.setErrOnce.Do(func() {
		proto.err = err
		close(proto.done)
	})
}

func (proto *Protocal) Wait() error {
	<-proto.done
	return proto.err
}
