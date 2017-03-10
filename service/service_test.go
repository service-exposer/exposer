package service

import (
	"errors"
	"net"
	"testing"
)

func TestService(t *testing.T) {
	service := newService("test")
	if service.Name() != "test" {
		t.Fatal("expect", "test", "got", service)
	}

	err := service.Attribute().View(func(attr Attribute) error {
		if attr.HTTP.Is != false {
			return errors.New("attr.HTTP.Is != false")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	func() {
		err := service.Close()
		if err == nil {
			t.Fatal("expect err")
		}
	}()

	func() {
		_, err := service.Open()
		if err == nil {
			t.Fatal("expect err")
		}
	}()

	closeCalled := false
	service.setCloseFunc(func() error {
		closeCalled = true
		return nil
	})

	service.Close()
	if !closeCalled {
		t.Fatal("expect", "closeCalled", "got", "!closeCalled")
	}

	openCalled := false
	service.setOpenFunc(func() (net.Conn, error) {
		openCalled = true
		return nil, nil
	})

	service.Open()
	if !openCalled {
		t.Fatal("expect", "openCalled", "got", "!openCalled")
	}
}
