package service

import "sync"

type Attribute struct {
	HTTP struct {
		Is   bool   `json:",omitempty"`
		Host string `json:",omitempty"`
	} `json:",omitempty"`
}

type SafedAttribute struct {
	mu   *sync.RWMutex
	attr Attribute
}

func NewSafedAttribute(attr *Attribute) *SafedAttribute {
	return &SafedAttribute{
		mu:   new(sync.RWMutex),
		attr: *attr,
	}
}

func (safed *SafedAttribute) View(fn func(attr Attribute) error) error {
	safed.mu.RLock()
	defer safed.mu.RUnlock()

	return fn(safed.attr)
}

func (safed *SafedAttribute) Update(fn func(attr *Attribute) error) error {
	safed.mu.Lock()
	defer safed.mu.Unlock()

	return fn(&safed.attr)
}
