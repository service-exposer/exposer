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
	if s == nil {
		return ""
	}
	return s.name
}

func (s *Service) Attribute() *SafedAttribute {
	if s == nil {
		return nil
	}
	return s.attr
}

func (s *Service) Open() (net.Conn, error) {
	if s == nil {
		return nil, errors.NotFoundf("service")
	}
	if s.openFn == nil {
		return nil, errors.Errorf("service %q is not ready", s.Name())
	}
	conn, err := s.openFn()
	return conn, errors.Annotatef(err, "Open %q", s.Name())
}
func (s *Service) Close() error {
	if s == nil {
		return errors.NotFoundf("service")
	}
	if s.closeFn == nil {
		return errors.Errorf("service %q is not ready", s.Name())
	}
	return errors.Annotatef(s.closeFn(), "Close %q", s.Name())
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
