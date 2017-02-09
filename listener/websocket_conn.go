package listener

import (
	"net"

	"github.com/gorilla/websocket"
)

type websocketConn struct {
	net.Conn
}

func NewWebsocketConn(conn *websocket.Conn) net.Conn {
	return &websocketConn{
		Conn: conn.UnderlyingConn(),
	}
}
