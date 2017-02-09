package listener

import (
	"errors"
	"net"

	"github.com/gorilla/websocket"
)

type websocketListener struct {
	accepts <-chan *websocket.Conn
	closeFn func() error
	addr    net.Addr
}

func Websocket(accepts <-chan *websocket.Conn, closeFn func() error, addr net.Addr) net.Listener {
	return &websocketListener{
		accepts: accepts,
		closeFn: closeFn,
		addr:    addr,
	}
}

func (ln *websocketListener) Accept() (net.Conn, error) {
	ws, ok := <-ln.accepts
	if !ok {
		return nil, errors.New("websocket listener closed")
	}

	return NewWebsocketConn(ws), nil
}

func (ln *websocketListener) Close() error {
	return ln.closeFn()
}
func (ln *websocketListener) Addr() net.Addr {
	return ln.addr
}
