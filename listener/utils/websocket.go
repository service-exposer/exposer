package utils

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

func WebsocketListener(network, addr string) (net.Listener, error) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	mutex := &sync.Mutex{}
	accepts := make(chan *websocket.Conn)
	closed := false

	server := http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ws, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			mutex.Lock()
			defer mutex.Unlock()

			if closed {
				w.WriteHeader(500)
				return
			}

			accepts <- ws
		}),
	}

	closeFn := func() error {
		mutex.Lock()
		defer mutex.Unlock()

		if !closed {
			close(accepts)
			closed = true
			ln.Close()
		}

		return nil
	}

	go func() {
		server.Serve(ln)
		closeFn()
	}()

	return listener.Websocket(accepts, closeFn, ln.Addr()), nil
}

func DialWebsocket(url string) (net.Conn, error) {
	ws, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	return listener.NewWebsocketConn(ws), nil
}
