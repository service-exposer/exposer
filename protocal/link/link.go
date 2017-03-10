package link

import (
	"encoding/json"
	"net"
	"sync"

	"github.com/juju/errors"
	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/service"
)

const (
	CMD_LINK       = "link"
	CMD_LINK_REPLY = "link:reply"
)

var (
	ErrServiceIsNotExist = errors.New("service is not exist")
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
				return errors.Trace(err)
			}

			service := router.Get(req.Name)
			if service == nil {
				proto.Reply(CMD_LINK_REPLY, &Reply{
					OK:  false,
					Err: ErrServiceIsNotExist.Error(),
				})

				return errors.Annotatef(ErrServiceIsNotExist, "%q", req.Name)
			}

			err = proto.Reply(CMD_LINK_REPLY, &Reply{
				OK: true,
			})
			if err != nil {
				return errors.Trace(err)
			}

			return func() (err error) {
				/*
					defer func() {
							// recover for service.Open,if you are sure that will not panic
							// just delete this recover defer
							if r := recover(); r != nil {
									err = errors.New(fmt.Sprint("panic:", r))
							}
					}()
				*/

				session := proto.Multiplex(false)
				for {
					remote, err := session.Accept()
					if err != nil {
						return errors.Trace(err)
					}

					local, err := service.Open()
					if err != nil {
						remote.Close()
						return errors.Trace(err)
					}

					go exposer.Forward(remote, local)
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
				return errors.Trace(err)
			}

			if !reply.OK {
				return errors.New(reply.Err)
			}

			session := proto.Multiplex(true)

			errch := make(chan error, 1)
			wg := new(sync.WaitGroup)

			wg.Add(1)
			go func() {
				defer wg.Done()

				// detect session closed
				// in fact,it's detecting underly conn closed
				// so,it will immediately reactive while
				// connection to server is closed
				err, _, _ := session.Wait()
				errch <- errors.Trace(err)
				ln.Close()
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()

				for {
					local, err := ln.Accept()
					if err != nil {
						errch <- errors.Trace(err)
						session.Close()
						return
					}
					remote, err := session.Open()
					if err != nil {
						errch <- errors.Trace(err)
						return
					}

					go exposer.Forward(remote, local)
				}
			}()

			go func() {
				wg.Wait()
				close(errch)
			}()

			defer func() {
				go func() {
					for range errch {
					}
				}()
			}()

			return <-errch
		}
		return errors.New("unknow cmd: " + cmd)
	}
}
