package service

import (
	"net"
	"sync"
)

type Service struct {
	name string
	attr *Attribute

	setOpenFuncOnce *sync.Once
	openFn          func() (net.Conn, error)

	setCloseFuncOnce *sync.Once
	closeFn          func() error
}

func newService(name string) *Service {
	return &Service{
		name:             name,
		attr:             NewAttribute(),
		setOpenFuncOnce:  new(sync.Once),
		openFn:           nil,
		setCloseFuncOnce: new(sync.Once),
		closeFn:          nil,
	}
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) Attribute() *Attribute {
	return s.attr
}

func (s *Service) Open() (net.Conn, error) {
	return s.openFn()
}
func (s *Service) Close() error {
	return s.closeFn()
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
