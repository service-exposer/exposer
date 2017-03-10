package route

import (
	"encoding/json"

	"github.com/juju/errors"
	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/protocal/expose"
	"github.com/service-exposer/exposer/protocal/forward"
	"github.com/service-exposer/exposer/protocal/keepalive"
	"github.com/service-exposer/exposer/protocal/link"
	"github.com/service-exposer/exposer/service"
)

const (
	CMD_ROUTE       = "route"
	CMD_ROUTE_REPLY = "route:reply"
)

type Type string

const (
	KeepAlive Type = "keepalive"
	Expose    Type = "expose"
	Link      Type = "link"
	Forward   Type = "forward"
)

var (
	ErrNotSupportedType = errors.New("not supported type")
)

type Reply struct {
	OK  bool
	Err string
}

type RouteReq struct {
	Type Type
}

func ServerSide(router *service.Router) exposer.HandshakeHandleFunc {
	keepaliveFn := keepalive.ServerSide(0)
	exposeFn := expose.ServerSide(router)
	linkFn := link.ServerSide(router)
	forwardFn := forward.ServerSide()

	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_ROUTE:
			var req RouteReq
			err := json.Unmarshal(details, &req)
			if err != nil {
				return errors.Trace(err)
			}

			switch req.Type {
			case KeepAlive:
				err := proto.Reply(CMD_ROUTE_REPLY, &Reply{
					OK: true,
				})
				if err != nil {
					return errors.Trace(err)
				}

				proto.On = keepaliveFn
			case Expose:
				err := proto.Reply(CMD_ROUTE_REPLY, &Reply{
					OK: true,
				})
				if err != nil {
					return errors.Trace(err)
				}

				proto.On = exposeFn
			case Link:
				err := proto.Reply(CMD_ROUTE_REPLY, &Reply{
					OK: true,
				})
				if err != nil {
					return errors.Trace(err)
				}

				proto.On = linkFn
			case Forward:
				err := proto.Reply(CMD_ROUTE_REPLY, &Reply{
					OK: true,
				})
				if err != nil {
					return errors.Trace(err)
				}

				proto.On = forwardFn
			default:
				err := errors.Annotatef(ErrNotSupportedType, "%q", req.Type)
				proto.Reply(CMD_ROUTE_REPLY, &Reply{
					OK:  false,
					Err: err.Error(),
				})

				return errors.Trace(err)
			}

			return nil
		}

		return errors.New("unknow cmd: " + cmd)
	}
}

func ClientSide(nextHandleFunc exposer.HandshakeHandleFunc, nextCmd string, nextDetails interface{}) exposer.HandshakeHandleFunc {
	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		switch cmd {
		case CMD_ROUTE_REPLY:
			var reply Reply
			err := json.Unmarshal(details, &reply)
			if err != nil {
				return errors.Trace(err)
			}

			if !reply.OK {
				return errors.New(reply.Err)
			}

			proto.On = nextHandleFunc
			return proto.Reply(nextCmd, nextDetails)
		}

		return errors.New("unknow cmd: " + cmd)
	}
}
