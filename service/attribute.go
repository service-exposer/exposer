package service

import (
	"sync"

	"github.com/juju/errors"
)

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

	return errors.Annotate(fn(safed.attr), "View")
}

func (safed *SafedAttribute) Update(fn func(attr *Attribute) error) error {
	safed.mu.Lock()
	defer safed.mu.Unlock()

	return errors.Annotate(fn(&safed.attr), "Update")
}
