package service

import (
	"net"
	"testing"
)

func TestService(t *testing.T) {
	service := newService("test")
	if service.Name() != "test" {
		t.Fatal("expect", "test", "got", service)
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				return
			}

			t.Fatal("expect panic")
		}()
		service.Close()
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				return
			}

			t.Fatal("expect panic")
		}()
		service.Open()
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
