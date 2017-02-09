package exposer

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestServe(t *testing.T) {
	const (
		CMD_ECHO_REQ   = "echo:req"
		CMD_ECHO_REPLY = "echo:reply"
	)
	type echoReq struct {
		Message string
	}

	type echoReply struct {
		Message string
	}

	ln, dial := NewPipeListener()
	go Serve(ln, func(conn net.Conn) ProtocalHandler {
		proto := NewProtocal(conn)
		proto.On = func(proto *Protocal, cmd string, details []byte) error {
			switch cmd {
			case CMD_ECHO_REQ:
				var req echoReq
				err := json.Unmarshal(details, &req)
				if err != nil {
					return err
				}

				return proto.Reply(CMD_ECHO_REPLY, &echoReply{
					Message: req.Message,
				})
			}
			return nil
		}

		return proto
	})

	// wait server setup
	time.Sleep(time.Second * 1)

	conn, err := dial()
	if err != nil {
		t.Fatal(err)
	}

	echo := make(chan string)
	proto := NewProtocal(conn)
	proto.On = func(proto *Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_ECHO_REPLY:
			var reply echoReply
			err := json.Unmarshal(details, &reply)
			if err != nil {
				return err
			}

			echo <- reply.Message
			close(echo)
		}
		return nil
	}

	expect := "exposer test"
	go proto.Request(CMD_ECHO_REQ, &echoReq{
		Message: expect,
	})

	select {
	case msg := <-echo:
		if msg != expect {
			t.Fatal("expect", expect, "got", msg)
		}

	case <-time.After(time.Second * 1):
		t.Fatal("timeout")
	}
}
