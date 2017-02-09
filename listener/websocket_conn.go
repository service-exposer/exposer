package listener

import (
	"bytes"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type websocketConn struct {
	conn    *websocket.Conn
	readbuf *bytes.Buffer
}

func NewWebsocketConn(conn *websocket.Conn) net.Conn {
	return &websocketConn{
		conn:    conn,
		readbuf: &bytes.Buffer{},
	}
}

func (ws *websocketConn) Read(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	if ws.readbuf.Len() > 0 {
		return ws.readbuf.Read(b)
	}

	for {
		msgtype, payload, err := ws.conn.ReadMessage()
		if msgtype == websocket.BinaryMessage {
			if len(payload) > len(b) {
				_, err := ws.readbuf.Write(payload[:len(b)])
				if err != nil {
					return 0, err
				}
			}

			n = copy(b, payload)
			return n, nil

		}
		if err != nil {
			return 0, err
		}
	}
}
func (ws *websocketConn) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	err = ws.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}
func (ws *websocketConn) Close() error {
	return ws.conn.Close()
}
func (ws *websocketConn) LocalAddr() net.Addr {
	return ws.conn.LocalAddr()
}
func (ws *websocketConn) RemoteAddr() net.Addr {
	return ws.conn.RemoteAddr()
}
func (ws *websocketConn) SetDeadline(t time.Time) error {
	return ws.conn.UnderlyingConn().SetDeadline(t)
}
func (ws *websocketConn) SetReadDeadline(t time.Time) error {
	return ws.conn.UnderlyingConn().SetReadDeadline(t)
}
func (ws *websocketConn) SetWriteDeadline(t time.Time) error {
	return ws.conn.UnderlyingConn().SetWriteDeadline(t)
}
