package service

import (
	"errors"
	"net"
	"sync"
)

var (
	ErrServiceExist = errors.New("Register: service name exist")
)

type Router struct {
	mu     *sync.Mutex
	routes map[string]*Service
}

func NewRouter() *Router {
	return &Router{
		mu:     new(sync.Mutex),
		routes: make(map[string]*Service),
	}
}

func (r *Router) Prepare(name string) error {
	if name == "" {
		return ErrServiceExist
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exist := r.routes[name]; exist {
		return ErrServiceExist
	}

	r.routes[name] = newService(name)
	return nil
}
func (r *Router) Add(name string, openFn func() (net.Conn, error),
	closeFn func() error) bool {
	if openFn == nil {
		panic("paramater openFn func() (net.Conn,error) is nil")
	}
	if closeFn == nil {
		panic("paramater closeFn func() error is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	service := r.routes[name]
	if service == nil {
		return false
	}

	service.setOpenFunc(openFn)
	service.setCloseFunc(closeFn)
	return true
}

func (r *Router) Get(name string) *Service {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.routes[name]
}

func (r *Router) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	service, exist := r.routes[name]
	if !exist {
		return
	}

	delete(r.routes, name)

	if service != nil && service.closeFn != nil {
		service.Close()
	}
}
