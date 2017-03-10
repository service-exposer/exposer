package protocal

import "net"

type ProtocalHandler interface {
	Handle()
	Request(cmd string, details interface{})
}

type NewProtocalHandler func(net.Conn) ProtocalHandler

func Serve(ln net.Listener, newHandler NewProtocalHandler) {
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// TODO: handle error
			return
		}

		go newHandler(conn).Handle()
	}
}
