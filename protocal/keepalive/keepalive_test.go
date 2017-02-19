package keepalive

import (
	"net"
	"testing"
	"time"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener"
)

func Test_keepalive(t *testing.T) {
	func() {
		cmds := make(chan string)

		ln, dial := listener.Pipe()
		go exposer.Serve(ln, func(conn net.Conn) exposer.ProtocalHandler {
			proto := exposer.NewProtocal(conn)
			handlefn := ServerSide(200 * time.Millisecond)

			proto.On = func(proto *exposer.Protocal, cmd string, details []byte) error {
				cmds <- cmd
				if cmd == EVENT_TIMEOUT {
					close(cmds)
				}
				return handlefn(proto, cmd, details)
			}

			return proto
		})

		conn, err := dial()
		if err != nil {
			t.Fatal(err)
		}

		proto := exposer.NewProtocal(conn)

		handlefn := ClientSide(300 * time.Millisecond)
		proto.On = func(proto *exposer.Protocal, cmd string, details []byte) error {
			cmds <- cmd
			return handlefn(proto, cmd, details)
		}
		go proto.Request(CMD_PING, nil)

		var cmd string
		cmd = <-cmds
		if cmd != CMD_PING {
			t.Fatal("expect", CMD_PING, "got", cmd)
		}
		cmd = <-cmds
		if cmd != CMD_PONG {
			t.Fatal("expect", CMD_PONG, "got", cmd)
		}
		cmd = <-cmds
		if cmd != EVENT_TIMEOUT {
			t.Fatal("expect", EVENT_TIMEOUT, "got", cmd)
		}
		_, ok := <-cmds
		if ok {
			t.Fatal("expect", "!ok", "got", "ok")
		}
	}()
}
