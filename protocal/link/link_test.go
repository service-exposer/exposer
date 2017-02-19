package link

import (
	"io"
	"net"
	"testing"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener"
	"github.com/service-exposer/exposer/service"
)

func Test_lisk(t *testing.T) {
	router := service.NewRouter()
	router.Prepare("test")

	accept := make(chan net.Conn, 2)
	ok := router.Add("test", func() (conn net.Conn, err error) {
		c1, c2 := net.Pipe()
		accept <- c1
		return c2, nil
	}, func() error {
		return nil
	})
	if !ok {
		t.Fatal("expect ok got !ok")
	}

	ln, dial := listener.Pipe()

	go exposer.Serve(ln, func(conn net.Conn) exposer.ProtocalHandler {
		proto := exposer.NewProtocal(conn)
		proto.On = ServerSide(router)
		return proto
	})

	conn, err := dial()
	if err != nil {
		t.Fatal(err)
	}

	lnLocal, dialLocal := listener.Pipe()

	proto := exposer.NewProtocal(conn)
	proto.On = ClientSide(lnLocal)
	go proto.Request(CMD_LINK, &LinkReq{
		Name: "test",
	})

	c1, err := dialLocal()
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		c1.Write([]byte("hello"))
		c1.Close()
	}()

	c2 := <-accept

	data := make([]byte, 5)
	_, err = io.ReadAtLeast(c2, data, len(data))
	if string(data) != "hello" {
		t.Fatal("expect", "hello", "got", string(data))
	}
}
