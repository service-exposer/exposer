package forward

import (
	"encoding/json"
	"errors"
	"net"

	"github.com/service-exposer/exposer"
)

const (
	CMD_FORWARD       = "forward"
	CMD_FORWARD_REPLY = "forward:reply"
)

type Reply struct {
	OK  bool
	Err string
}

type Forward struct {
	Network string
	Address string
}

func ServerSide() exposer.HandshakeHandleFunc {
	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_FORWARD:
			var forward Forward
			err := json.Unmarshal(details, &forward)
			if err != nil {
				return err
			}

			conn, err := net.Dial(forward.Network, forward.Address)
			if err != nil {
				proto.Reply(CMD_FORWARD_REPLY, &Reply{
					OK:  false,
					Err: err.Error(),
				})
				return err
			}
			conn.Close()

			err = proto.Reply(CMD_FORWARD_REPLY, &Reply{
				OK: true,
			})
			if err != nil {
				return err
			}

			/*
				ln := proto.Multiplex(false)
				defer ln.Close()

				for {
					local_conn, err := ln.Accept()
					if err != nil {
						return err
					}
					defer local_conn.Close()


					remote_conn, err := net.Dial(forward.Network, forward.Address)
					if err != nil {
						return err
					}
					defer remote_conn.Close()

					go func() {
						wg := &sync.WaitGroup{}
						wg.Add(2)
						go func() {
							defer wg.Done()
							io.Copy(local_conn, remote_conn)
						}()

						go func() {
							defer wg.Done()
							io.Copy(remote_conn, local_conn)
						}()
						wg.Wait()
					}()
				}
			*/
			exposer.Serve(proto.Multiplex(false), func(conn net.Conn) exposer.ProtocalHandler {
				proto := exposer.NewProtocal(conn)
				proto.On = func(proto *exposer.Protocal, cmd string, details []byte) error {
					err := proto.Reply("", nil)
					if err != nil {
						return err
					}

					conn, err := net.Dial(forward.Network, forward.Address)
					if err != nil {
						return err
					}

					proto.Forward(conn)
					return nil
				}
				return proto
			})

		}
		return errors.New("unknow cmd")
	}

}

func ClientSide(ln net.Listener) exposer.HandshakeHandleFunc {
	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_FORWARD_REPLY:

			var reply Reply
			err := json.Unmarshal(details, &reply)
			if err != nil {
				return err
			}

			if !reply.OK {
				return errors.New(reply.Err)
			}

			session := proto.Multiplex(true)

			for {
				local_conn, err := ln.Accept()
				if err != nil {
					return err
				}

				remote_conn, err := session.Open()
				if err != nil {
					return err
				}

				/*
					go func() {
						wg := &sync.WaitGroup{}
						wg.Add(2)
						go func() {
							defer wg.Done()
							io.Copy(local_conn, remote_conn)
						}()

						go func() {
							defer wg.Done()
							io.Copy(remote_conn, local_conn)
						}()
						wg.Wait()
					}()
				*/

				proto_forward := exposer.NewProtocal(remote_conn)
				proto_forward.On = func(proto *exposer.Protocal, cmd string, details []byte) error {
					proto.Forward(local_conn)
					return nil
				}

				go proto_forward.Request("", nil)
			}
		default:
			return errors.New("unknow cmd")
		}
	}

}
