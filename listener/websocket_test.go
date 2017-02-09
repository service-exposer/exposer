package listener

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestWebsocket(t *testing.T) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	accepts := make(chan *websocket.Conn, 16)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		accepts <- ws
	}))
	defer ts.Close()

	go func() {
		ln := Websocket(accepts, func() error {
			return nil
		}, ts.Listener.Addr())

		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}

			go func() {
				defer conn.Close()

				io.Copy(conn, conn)
			}()
		}
	}()

	dialer := websocket.Dialer{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	ws, _, err := dialer.Dial(strings.Replace(ts.URL, "http", "ws", 1), nil)
	if err != nil {
		t.Fatal(err)
	}

	conn := NewWebsocketConn(ws)
	conn.Write([]byte("hello"))

	readbuf := make([]byte, 5)
	_, err = io.ReadAtLeast(conn, readbuf, 5)
	if err != nil {
		t.Fatal(err)
	}

	if string(readbuf) != "hello" {
		t.Fatal("expect", "hello", "got", string(readbuf))
	}
}
