package exposer

import (
	"encoding/json"
	"io"
	"net"
	"sync"
	"time"

	"github.com/inconshreveable/muxado"
	"github.com/juju/errors"
)

type HandshakeHandleFunc func(proto *Protocal, cmd string, details []byte) error
type Protocal struct {
	parent *Protocal

	conn             net.Conn
	isHandshakeDone  bool
	handshakeDecoder *json.Decoder
	done             chan struct{}

	eventbus            chan HandshakeIncoming
	eventbusClosed      bool
	eventbusClosedMutex *sync.RWMutex

	setErrOnce *sync.Once
	err        error

	// handle handshake
	mutex_On *sync.Mutex
	On       HandshakeHandleFunc
}

func NewProtocal(conn net.Conn) *Protocal {
	return &Protocal{
		parent: nil,

		conn:             conn,
		isHandshakeDone:  false,
		handshakeDecoder: json.NewDecoder(conn),
		done:             make(chan struct{}),

		eventbus:            make(chan HandshakeIncoming),
		eventbusClosed:      false,
		eventbusClosedMutex: new(sync.RWMutex),

		setErrOnce: new(sync.Once),
		err:        nil,

		mutex_On: new(sync.Mutex),
		On:       nil,
	}
}

func NewProtocalWithParent(parent *Protocal, conn net.Conn) *Protocal {
	proto := NewProtocal(conn)
	proto.parent = parent
	return proto
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
		return errors.Trace(err)
	}

	_, err = proto.conn.Write(data)
	return errors.Trace(err)
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

const (
	wait_time_before_close = 8 * time.Second // wait other conn read data
)

func Forward(c1, c2 net.Conn) {
	go func() {
		io.Copy(c1, c2)
		time.Sleep(wait_time_before_close)
		c1.Close()
	}()
	io.Copy(c2, c1)
	time.Sleep(wait_time_before_close)
	c2.Close()
}

func (proto *Protocal) Forward(conn net.Conn) {
	proto.isHandshakeDone = true

	go func() {
		io.Copy(conn, io.MultiReader(proto.handshakeDecoder.Buffered(), proto.conn))
		time.Sleep(wait_time_before_close)
		conn.Close()
	}()
	io.Copy(proto.conn, conn)
	time.Sleep(wait_time_before_close)
	proto.conn.Close()
	/*
		go io.Copy(conn, io.MultiReader(proto.handshakeDecoder.Buffered(), proto.conn))
		io.Copy(proto.conn, conn)
	*/
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
		return errors.Trace(err)
	}

	var handshake HandshakeIncoming
	err = json.Unmarshal(data, &handshake)
	if err != nil {
		return errors.Trace(err)
	}

	/*
		defer func() {
			// chan proto.eventbus maybe closed,so use recover
			if r := recover(); r != nil {
				err = errors.New("conn closed")
			}
		}()
	*/
	proto.eventbusClosedMutex.RLock()
	defer proto.eventbusClosedMutex.RUnlock()
	if proto.eventbusClosed {
		return errors.New("conn closed")
	}
	proto.eventbus <- handshake
	return nil
}

func (proto *Protocal) Handle() {
	defer proto.conn.Close()
	defer proto.Shutdown(errors.New("ok"))

	if proto.On == nil {
		panic("not set Protocal.On")
	}

	handleHandshake := func(proto *Protocal, handshake HandshakeIncoming) bool {
		proto.mutex_On.Lock()
		defer proto.mutex_On.Unlock()

		if proto.isShutdown() {
			return false
		}

		err := proto.On(proto, handshake.Command, handshake.Details)
		if err != nil {
			proto.Shutdown(errors.Trace(err))
			return false
		}
		return true
	}

	go func() {
		defer proto.conn.Close()

		isDone := false
		for handshake := range proto.eventbus {
			// recv all messages while proto.eventbus chan is closed
			if isDone {
				continue
			}

			ok := handleHandshake(proto, handshake)
			if !ok {
				isDone = true
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
		ok := handleHandshake(proto, handshake)
		if !ok {
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

		proto.eventbusClosedMutex.Lock()
		close(proto.eventbus)
		proto.eventbusClosed = true
		proto.eventbusClosedMutex.Unlock()

		if proto.parent != nil {
			proto.parent.Shutdown(err)
		}
	})
}

func (proto *Protocal) isShutdown() bool {
	select {
	case <-proto.done:
		return true
	default:
		return false
	}
}

func (proto *Protocal) Wait() error {
	<-proto.done
	return errors.Trace(proto.err)
}
