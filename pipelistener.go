package exposer

import (
	"errors"
	"net"
	"sync"
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
