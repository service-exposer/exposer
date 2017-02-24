package service

import "sync"

type Attribute struct {
	mu   *sync.RWMutex
	HTTP struct {
		Is   bool
		Host string
	}
}

func NewAttribute() *Attribute {
	return &Attribute{
		mu: new(sync.RWMutex),
	}
}

func (attr *Attribute) View(fn func(attr *Attribute) error) error {
	attr.mu.RLock()
	defer attr.mu.RUnlock()

	return fn(attr)
}

func (attr *Attribute) Update(fn func(attr *Attribute) error) error {
	attr.mu.Lock()
	defer attr.mu.Unlock()

	return fn(attr)
}
