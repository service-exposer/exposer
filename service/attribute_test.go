package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAttribute_View(t *testing.T) {
	attr := newAttribute()
	err := attr.View(func(attr *Attribute) error {
		if attr.HTTP.Is != false {
			t.Fatal("want", false)
		}
		if attr.HTTP.Host != "" {
			t.Fatal(attr.HTTP.Host, "want", "")
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	err = attr.View(func(attr *Attribute) error {
		return errors.New("test error bubbling")
	})

	if err == nil {
		t.Fatal("expect not nil error")
	}
}

func TestAttribute_Update(t *testing.T) {
	attr := newAttribute()
	err := attr.Update(func(attr *Attribute) error {
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	err = attr.Update(func(attr *Attribute) error {
		return errors.New("test error bubbling")
	})

	if err == nil {
		t.Fatal("expect not nil error")
	}

	err = attr.Update(func(attr *Attribute) error {
		attr.HTTP.Is = true
		attr.HTTP.Host = "hostname.test"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if attr.HTTP.Is != true {
		t.Fatal("want", true)
	}

	if attr.HTTP.Host != "hostname.test" {
		t.Fatal(attr.HTTP.Host, "want", "hostname.test")
	}
}

func TestAttribute_UpdateAndView(t *testing.T) {
	attr := newAttribute()
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*200)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			attr.Update(func(attr *Attribute) error {
				attr.HTTP.Is = true
				attr.HTTP.Host = "hostname.test"
				return nil
			})
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
		attr.View(func(attr *Attribute) error {
			if attr.HTTP.Is != true {
				return errors.New("attr.HTTP.Is != true")
			}
			if attr.HTTP.Host != "hostname.test" {
				return errors.New("attr.HTTP.Host != hostname.test; got " + attr.HTTP.Host)
			}
			return nil
		})
	}()
}
