package route

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/service-exposer/exposer/listener"
	"github.com/service-exposer/exposer/protocal"
	"github.com/service-exposer/exposer/protocal/expose"
	"github.com/service-exposer/exposer/protocal/forward"
	"github.com/service-exposer/exposer/protocal/keepalive"
	"github.com/service-exposer/exposer/protocal/link"
	"github.com/service-exposer/exposer/service"
)

func Test_route(t *testing.T) {
	ln, dial := listener.Pipe()

	go protocal.Serve(ln, func(conn net.Conn) protocal.ProtocalHandler {
		proto := protocal.NewProtocal(conn)
		proto.On = ServerSide(service.NewRouter())
		return proto
	})

	func() {
		conn, err := dial()
		if err != nil {
			t.Fatal(err)
		}

		cmds := make(chan string)

		proto := protocal.NewProtocal(conn)
		handlefn := keepalive.ClientSide(0, 100*time.Millisecond)
		proto.On = ClientSide(func(proto *protocal.Protocal, cmd string, details []byte) error {
			cmds <- cmd
			return handlefn(proto, cmd, details)
		}, keepalive.CMD_PING, nil)
		go proto.Request(CMD_ROUTE, &RouteReq{
			Type: KeepAlive,
		})

		cmd := <-cmds
		if cmd != keepalive.CMD_PONG {
			t.Fatal("expect", keepalive.CMD_PONG, "got", cmd)
		}
	}()

	func() {
		conn, err := dial()
		if err != nil {
			t.Fatal(err)
		}

		cmds := make(chan string)

		proto := protocal.NewProtocal(conn)
		handlefn := expose.ClientSide(func() (net.Conn, error) {
			return nil, errors.New("test dial")
		})
		proto.On = ClientSide(func(proto *protocal.Protocal, cmd string, details []byte) error {
			cmds <- cmd
			return handlefn(proto, cmd, details)
		}, expose.CMD_EXPOSE, &expose.ExposeReq{})

		go proto.Request(CMD_ROUTE, &RouteReq{
			Type: Expose,
		})

		cmd := <-cmds
		if cmd != expose.CMD_EXPOSE_REPLY {
			t.Fatal("expect", expose.CMD_EXPOSE_REPLY, "got", cmd)
		}
	}()

	func() {
		conn, err := dial()
		if err != nil {
			t.Fatal(err)
		}

		cmds := make(chan string)

		proto := protocal.NewProtocal(conn)

		ln, _ := listener.Pipe()
		handlefn := link.ClientSide(ln)

		proto.On = ClientSide(func(proto *protocal.Protocal, cmd string, details []byte) error {
			cmds <- cmd
			return handlefn(proto, cmd, details)
		}, link.CMD_LINK, &link.LinkReq{})

		go proto.Request(CMD_ROUTE, &RouteReq{
			Type: Link,
		})

		cmd := <-cmds
		if cmd != link.CMD_LINK_REPLY {
			t.Fatal("expect", link.CMD_LINK_REPLY, "got", cmd)
		}
	}()

	func() {
		conn, err := dial()
		if err != nil {
			t.Fatal(err)
		}

		cmds := make(chan string)

		proto := protocal.NewProtocal(conn)

		ln, _ := listener.Pipe()
		handlefn := forward.ClientSide(ln)

		proto.On = ClientSide(func(proto *protocal.Protocal, cmd string, details []byte) error {
			cmds <- cmd
			return handlefn(proto, cmd, details)
		}, forward.CMD_FORWARD, &forward.Forward{})

		go proto.Request(CMD_ROUTE, &RouteReq{
			Type: Forward,
		})

		cmd := <-cmds
		if cmd != forward.CMD_FORWARD_REPLY {
			t.Fatal("expect", forward.CMD_FORWARD_REPLY, "got", cmd)
		}
	}()
}
