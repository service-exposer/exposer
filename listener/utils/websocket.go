package utils

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/juju/errors"
	"github.com/service-exposer/exposer/listener"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 15 * time.Second,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var dialer = websocket.Dialer{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 15 * time.Second,
}

func WebsocketHandlerListener(addr net.Addr) (net.Listener, http.Handler, error) {
	var (
		mutex  = new(sync.Mutex)
		closed = false
	)
	accepts := make(chan *websocket.Conn)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		if closed {
			return
		}

		accepts <- ws
	})

	closeFn := func() error {
		mutex.Lock()
		defer mutex.Unlock()

		if !closed {
			close(accepts)
			closed = true
		}

		return nil
	}

	return listener.Websocket(accepts, closeFn, addr), handler, nil
}
func WebsocketListener(network, addr string) (net.Listener, error) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, errors.Trace(err)
	}

	wsln, handler, err := WebsocketHandlerListener(ln.Addr())
	if err != nil {
		return nil, errors.Trace(err)
	}

	server := http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      handler,
	}

	go func() {
		server.Serve(ln)
	}()

	return wsln, nil
}

func DialWebsocket(url string) (net.Conn, error) {
	ws, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return listener.NewWebsocketConn(ws), nil
}
