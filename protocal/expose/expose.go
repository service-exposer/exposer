package expose

import (
	"encoding/json"
	"net"

	"github.com/juju/errors"
	"github.com/service-exposer/exposer/protocal"
	"github.com/service-exposer/exposer/service"
)

const (
	CMD_EXPOSE       = "expose"
	CMD_EXPOSE_REPLY = "expose:reply"
)

type Reply struct {
	OK  bool
	Err string
}

type ExposeReq struct {
	Name string
	Attr service.Attribute
}

func ServerSide(router *service.Router) protocal.HandshakeHandleFunc {
	return func(proto *protocal.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_EXPOSE:
			var req ExposeReq
			err := json.Unmarshal(details, &req)
			if err != nil {
				return errors.Trace(err)
			}

			err = router.Prepare(req.Name)
			if err != nil {
				proto.Reply(CMD_EXPOSE_REPLY, &Reply{
					OK:  false,
					Err: err.Error(),
				})

				return errors.Trace(err)
			}
			defer router.Remove(req.Name)

			err = proto.Reply(CMD_EXPOSE_REPLY, &Reply{
				OK: true,
			})
			if err != nil {
				return errors.Trace(err)
			}

			session := proto.Multiplex(true)

			ok := router.Add(req.Name, session.Open, session.Close)
			if !ok {
				return errors.New("Router.Add failure")
			}
			router.Get(req.Name).Attribute().Update(func(attr *service.Attribute) error {
				*attr = req.Attr
				return nil
			})
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
func ClientSide(dial func() (net.Conn, error)) protocal.HandshakeHandleFunc {
	return func(proto *protocal.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_EXPOSE_REPLY:
			var reply Reply
			err := json.Unmarshal(details, &reply)
			if err != nil {
				return errors.Trace(err)
			}

			if !reply.OK {
				return errors.New(reply.Err)
			}

			session := proto.Multiplex(false)

			for {
				remote, err := session.Accept()
				if err != nil {
					return errors.Trace(err)
				}

				local, err := dial()
				if err != nil {
					remote.Close()
					continue
				}

				go protocal.Forward(remote, local)
			}
		}

		return errors.New("unknow cmd: " + cmd)
	}
}
