package expose

import (
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener"
	"github.com/service-exposer/exposer/service"
)

func Test_expose(t *testing.T) {
	router := service.NewRouter()
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

	accept := make(chan net.Conn, 2)
	proto := exposer.NewProtocal(conn)
	proto.On = ClientSide(func() (net.Conn, error) {
		c1, c2 := net.Pipe()
		accept <- c1
		return c2, nil
	})

	go proto.Request(CMD_EXPOSE, &ExposeReq{
		Name: "test",
	})

	time.Sleep(time.Millisecond * 10)

	c1, err := router.Get("test").Open()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		c1.Write([]byte("hello"))
		c1.Close()
	}()
	c2 := <-accept

	data, err := ioutil.ReadAll(c2)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "hello" {
		t.Fatal("expect hello got", string(data))
	}
}
