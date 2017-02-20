package auth

import (
	"net"
	"testing"
	"time"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener"
	"github.com/service-exposer/exposer/protocal/keepalive"
	"github.com/service-exposer/exposer/protocal/route"
	"github.com/service-exposer/exposer/service"
)

func Test_auth(t *testing.T) {
	ln, dial := listener.Pipe()

	authRes := make(chan bool)

	go exposer.Serve(ln, func(conn net.Conn) exposer.ProtocalHandler {
		proto := exposer.NewProtocal(conn)
		proto.On = ServerSide(service.NewRouter(), func(key string) bool {
			auth := key == "test"
			authRes <- auth
			return auth
		})
		return proto
	})

	func() {
		conn, err := dial()
		if err != nil {
			t.Fatal(err)
		}
		nextRoutes := make(chan NextRoute)

		proto := exposer.NewProtocal(conn)
		proto.On = ClientSide(nextRoutes)
		go proto.Request(CMD_AUTH, &AuthReq{
			Key: "",
		})

		res := <-authRes
		if res != false {
			t.Fatal("expect auth failure")
		}
	}()

	func() {
		conn, err := dial()
		if err != nil {
			t.Fatal(err)
		}
		nextRoutes := make(chan NextRoute, 2)

		proto := exposer.NewProtocal(conn)
		proto.On = ClientSide(nextRoutes)
		go proto.Request(CMD_AUTH, &AuthReq{
			Key: "test",
		})

		res := <-authRes
		if res != true {
			t.Fatal("expect auth correct")
		}

		keepaliveCmds_1 := make(chan string)
		nextRoutes <- NextRoute{
			Req: route.RouteReq{
				Type: route.KeepAlive,
			},
			HandleFunc: func() exposer.HandshakeHandleFunc {
				handlefn := keepalive.ClientSide(100 * time.Millisecond)

				return func(proto *exposer.Protocal, cmd string, details []byte) error {
					keepaliveCmds_1 <- cmd
					return handlefn(proto, cmd, details)
				}
			}(),
			Cmd: keepalive.CMD_PING,
		}

		keepaliveCmds_2 := make(chan string)
		nextRoutes <- NextRoute{
			Req: route.RouteReq{
				Type: route.KeepAlive,
			},
			HandleFunc: func() exposer.HandshakeHandleFunc {
				handlefn := keepalive.ClientSide(100 * time.Millisecond)

				return func(proto *exposer.Protocal, cmd string, details []byte) error {
					keepaliveCmds_2 <- cmd
					return handlefn(proto, cmd, details)
				}
			}(),
			Cmd: keepalive.CMD_PING,
		}

		cmd_1 := <-keepaliveCmds_1
		if cmd_1 != keepalive.CMD_PONG {
			t.Fatal("expect", keepalive.CMD_PONG, "got", cmd_1)
		}

		cmd_2 := <-keepaliveCmds_2
		if cmd_1 != keepalive.CMD_PONG {
			t.Fatal("expect", keepalive.CMD_PONG, "got", cmd_2)
		}

	}()

}
