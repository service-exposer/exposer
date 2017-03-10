package keepalive

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/juju/errors"
	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener"
)

type command struct {
	isClient bool
	isServer bool
	cmd      string
}

func (cmd *command) String() string {
	return fmt.Sprintf("%#v", cmd)
}

func Test_keepalive(t *testing.T) {
	ms := func(n int) time.Duration {
		return time.Duration(n) * time.Millisecond
	}

	test_keepalive := func(t *testing.T,
		server_timeout, server_delay, client_timeout, client_interval time.Duration,
	) <-chan *command {
		cmds := make(chan *command)

		ln, dial := listener.Pipe()
		go exposer.Serve(ln, func(conn net.Conn) exposer.ProtocalHandler {
			proto := exposer.NewProtocal(conn)
			handlefn := ServerSide(server_timeout)

			proto.On = func(proto *exposer.Protocal, cmd string, details []byte) error {
				time.Sleep(server_delay)
				cmds <- &command{
					isServer: true,
					cmd:      cmd,
				}
				return errors.Trace(handlefn(proto, cmd, details))
			}

			return proto
		})

		conn, err := dial()
		if err != nil {
			t.Fatal(err)
		}

		proto := exposer.NewProtocal(conn)

		handlefn := ClientSide(client_timeout, client_interval)
		proto.On = func(proto *exposer.Protocal, cmd string, details []byte) error {
			cmds <- &command{
				isClient: true,
				cmd:      cmd,
			}
			return handlefn(proto, cmd, details)
		}
		go proto.Request(CMD_PING, nil)

		var cmd *command
		cmd = <-cmds
		if cmd.cmd != CMD_PING || !cmd.isServer {
			t.Fatal("expect", CMD_PING, "& isServer", "got", cmd)
		}
		cmd = <-cmds
		if cmd.cmd != CMD_PONG || !cmd.isClient {
			t.Fatal("expect", CMD_PONG, "& isClient", "got", cmd)
		}
		return cmds
	}

	var cmds <-chan *command
	var cmd *command

	func() {
		cmds = test_keepalive(t, ms(60), ms(0), ms(60), ms(30))
		cmd = <-cmds
		if cmd.cmd != CMD_PING || !cmd.isServer {
			t.Fatal("expect", CMD_PING, "& isServer", "got", cmd)
		}
		cmd = <-cmds
		if cmd.cmd != CMD_PONG || !cmd.isClient {
			t.Fatal("expect", CMD_PONG, "& isClient", "got", cmd)
		}
	}()
	func() {
		cmds = test_keepalive(t, ms(50), ms(0), ms(100), ms(90))
		cmd = <-cmds
		if cmd.cmd != EVENT_TIMEOUT || !cmd.isServer {
			t.Fatal("expect", EVENT_TIMEOUT, "& isServer", "got", cmd)
		}
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expect panic")
			}
		}()
		cmds = test_keepalive(t, ms(30), ms(0), ms(30), ms(90))
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expect panic")
			}
		}()
		cmds = test_keepalive(t, ms(30), ms(0), ms(90), ms(90))
	}()

	func() {
		cmds = test_keepalive(t, ms(150), ms(90), ms(150), ms(90))
		cmd = <-cmds
		if cmd.cmd != EVENT_TIMEOUT || !cmd.isClient {
			t.Fatal("expect", EVENT_TIMEOUT, "& isClient", "got", cmd)
		}
	}()
}
