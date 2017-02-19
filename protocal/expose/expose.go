package expose

import (
	"encoding/json"
	"errors"
	"io"
	"net"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/service"
)

const (
	CMD_EXPOSE       = "expose"
	CMD_EXPOSE_REPLY = "expose:reply"
)

const ()

type Reply struct {
	OK  bool
	Err string
}

type ExposeReq struct {
	Name string
}

func ServerSide(router *service.Router) exposer.HandshakeHandleFunc {
	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_EXPOSE:
			var req ExposeReq
			err := json.Unmarshal(details, &req)
			if err != nil {
				return err
			}

			err = router.Prepare(req.Name)
			if err != nil {
				proto.Reply(CMD_EXPOSE_REPLY, &Reply{
					OK:  false,
					Err: err.Error(),
				})

				return err
			}
			defer router.Remove(req.Name)

			err = proto.Reply(CMD_EXPOSE_REPLY, &Reply{
				OK: true,
			})
			if err != nil {
				return err
			}

			session := proto.Multiplex(true)

			ok := router.Add(req.Name, session.Open, session.Close)
			if !ok {
				return errors.New("Router.Add failure")
			}
			defer func() {
				service := router.Get(req.Name)
				if service != nil {
					service.Close()
				}
			}()

			session.Wait()

			return nil
		}

		return errors.New("unknow cmd: " + cmd)

	}
}
func ClientSide(dial func() (net.Conn, error)) exposer.HandshakeHandleFunc {
	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_EXPOSE_REPLY:
			var reply Reply
			err := json.Unmarshal(details, &reply)
			if err != nil {
				return err
			}

			if !reply.OK {
				return errors.New(reply.Err)
			}

			session := proto.Multiplex(false)

			for {
				remote, err := session.Accept()
				if err != nil {
					return err
				}

				local, err := dial()
				if err != nil {
					remote.Close()
					continue
				}

				go func(remote, local net.Conn) { // forward
					defer remote.Close()
					defer local.Close()

					go io.Copy(remote, local)
					io.Copy(local, remote)
				}(remote, local)
			}
			return nil
		}

		return errors.New("unknow cmd: " + cmd)
	}
}
