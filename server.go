package exposer

import "net"

type TransportHandler interface {
	Handle()
	Request(cmd string, details interface{})
}

type NewTransportHandler func(net.Conn) TransportHandler

func Serve(ln net.Listener, newHandler NewTransportHandler) {
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
