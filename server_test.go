package exposer

import (
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

type pipeListener struct {
	*sync.Mutex
	accepts chan net.Conn
	conns   map[net.Conn]struct{}
	closed  bool
}

func NewPipeListener() (ln net.Listener, dial func() (net.Conn, error)) {
	pipeln := &pipeListener{
		Mutex:   &sync.Mutex{},
		accepts: make(chan net.Conn, 2),
		conns:   make(map[net.Conn]struct{}),
		closed:  false,
	}

	return pipeln, pipeln.Dial
}

func (ln *pipeListener) Dial() (net.Conn, error) {
	ln.Lock()
	defer ln.Unlock()

	if ln.closed {
		return nil, errors.New("closed")
	}

	c1, c2 := net.Pipe()

	ln.conns[c1] = struct{}{}
	ln.conns[c2] = struct{}{}

	ln.accepts <- c1

	return c2, nil
}

func (ln *pipeListener) Accept() (net.Conn, error) {
	c, ok := <-ln.accepts
	if !ok {
		return nil, errors.New("closed")
	}

	return c, nil
}

func (ln *pipeListener) Close() error {
	ln.Lock()
	defer ln.Unlock()

	if !ln.closed {
		close(ln.accepts)
		for c := range ln.conns {
			c.Close()
		}

		ln.closed = true
	}

	return nil
}

type pipeaddr struct{}

func (addr *pipeaddr) Network() string {
	return "pipe"
}
func (addr *pipeaddr) String() string {
	return "pipe"
}

func (ln *pipeListener) Addr() net.Addr {
	return &pipeaddr{}
}

func TestServe(t *testing.T) {
	const (
		CMD_ECHO_REQ   = "echo:req"
		CMD_ECHO_REPLY = "echo:reply"
	)
	type echoReq struct {
		Message string
	}

	type echoReply struct {
		Message string
	}

	ln, dial := NewPipeListener()
	go Serve(ln, func(conn net.Conn) TransportHandler {
		trans := NewTransport(conn)
		trans.On = func(trans *Transport, cmd string, details []byte) error {
			switch cmd {
			case CMD_ECHO_REQ:
				var req echoReq
				err := json.Unmarshal(details, &req)
				if err != nil {
					return err
				}

				return trans.Reply(CMD_ECHO_REPLY, &echoReply{
					Message: req.Message,
				})
			}
			return nil
		}

		return trans
	})

	// wait server setup
	time.Sleep(time.Second * 1)

	conn, err := dial()
	if err != nil {
		t.Fatal(err)
	}

	echo := make(chan string)
	trans := NewTransport(conn)
	trans.On = func(trans *Transport, cmd string, details []byte) error {
		switch cmd {
		case CMD_ECHO_REPLY:
			var reply echoReply
			err := json.Unmarshal(details, &reply)
			if err != nil {
				return err
			}

			echo <- reply.Message
			close(echo)
		}
		return nil
	}

	expect := "exposer test"
	go trans.Request(CMD_ECHO_REQ, &echoReq{
		Message: expect,
	})

	select {
	case msg := <-echo:
		if msg != expect {
			t.Fatal("expect", expect, "got", msg)
		}

	case <-time.After(time.Second * 1):
		t.Fatal("timeout")
	}
}
