package service

import (
	"net"
	"sync"

	"github.com/juju/errors"
)

type Service struct {
	mu      *sync.RWMutex
	name    string
	attr    *SafedAttribute
	openFn  func() (net.Conn, error)
	closeFn func() error
}

func newService(name string) *Service {
	return &Service{
		mu:      new(sync.RWMutex),
		name:    name,
		attr:    NewSafedAttribute(new(Attribute)),
		openFn:  nil,
		closeFn: nil,
	}
}

func (s *Service) Name() string {
	if s == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.name
}

func (s *Service) Attribute() *SafedAttribute {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.attr
}

func (s *Service) Open() (net.Conn, error) {
	if s == nil {
		return nil, errors.NotFoundf("service")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

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

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closeFn == nil {
		return errors.Errorf("service %q is not ready", s.Name())
	}
	return errors.Annotatef(s.closeFn(), "Close %q", s.Name())
}

func (s *Service) setOpenFunc(fn func() (net.Conn, error)) {
	if s == nil {
		panic("service is nil")
	}

	if fn == nil {
		panic("fn is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.openFn = fn
}

func (s *Service) setCloseFunc(fn func() error) {
	if s == nil {
		panic("service is nil")
	}

	if fn == nil {
		panic("fn is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.closeFn = fn
}
