package service

import (
	"net"
	"sync"

	"github.com/juju/errors"
)

type Service struct {
	name string
	attr *SafedAttribute

	setOpenFuncOnce *sync.Once
	openFn          func() (net.Conn, error)

	setCloseFuncOnce *sync.Once
	closeFn          func() error
}

func newService(name string) *Service {
	return &Service{
		name:             name,
		attr:             NewSafedAttribute(new(Attribute)),
		setOpenFuncOnce:  new(sync.Once),
		openFn:           nil,
		setCloseFuncOnce: new(sync.Once),
		closeFn:          nil,
	}
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) Attribute() *SafedAttribute {
	return s.attr
}

func (s *Service) Open() (net.Conn, error) {
	conn, err := s.openFn()
	return conn, errors.Annotatef(err, "Open %q", s)
}
func (s *Service) Close() error {
	return errors.Annotatef(s.closeFn(), "Close %q", s)
}

func (s *Service) setOpenFunc(fn func() (net.Conn, error)) {
	s.setOpenFuncOnce.Do(func() {
		s.openFn = fn
	})
}

func (s *Service) setCloseFunc(fn func() error) {
	s.setCloseFuncOnce.Do(func() {
		s.closeFn = fn
	})
}
