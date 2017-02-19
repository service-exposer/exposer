package link

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/service"
)

const (
	CMD_LINK       = "link"
	CMD_LINK_REPLY = "link:reply"
)

type Reply struct {
	OK  bool
	Err string
}

type LinkReq struct {
	Name string
}

func ServerSide(router *service.Router) exposer.HandshakeHandleFunc {
	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_LINK:
			var req LinkReq
			err := json.Unmarshal(details, &req)
			if err != nil {
				return err
			}

			service := router.Get(req.Name)
			if service == nil {
				err := errors.New(fmt.Sprint("service", req.Name, "is not exist"))
				proto.Reply(CMD_LINK_REPLY, &Reply{
					OK:  false,
					Err: err.Error(),
				})

				return err
			}

			err = proto.Reply(CMD_LINK_REPLY, &Reply{
				OK: true,
			})
			if err != nil {
				return err
			}

			return func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = errors.New(fmt.Sprint("panic:", r))
					}
				}()

				session := proto.Multiplex(false)
				for {
					remote, err := session.Accept()
					if err != nil {
						return err
					}

					local, err := service.Open()
					if err != nil {
						remote.Close()
						return err
					}

					go func() { // forward
						defer remote.Close()
						defer local.Close()

						go io.Copy(remote, local)
						io.Copy(local, remote)
					}()
				}
			}()
		}
		return errors.New("unknow cmd: " + cmd)
	}
}

func ClientSide(ln net.Listener) exposer.HandshakeHandleFunc {
	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_LINK_REPLY:
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
				local, err := ln.Accept()
				if err != nil {
					return err
				}
				remote, err := session.Open()
				if err != nil {
					return err
				}

				go func() { // forward
					defer remote.Close()
					defer local.Close()

					go io.Copy(remote, local)
					io.Copy(local, remote)
				}()
			}
			return nil
		}
		return errors.New("unknow cmd: " + cmd)
	}
}
