package service

import (
	"net"
	"testing"

	"github.com/juju/errors"
)

func TestRouter_Prepare(t *testing.T) {
	r := NewRouter()

	err := r.Prepare("")
	if errors.Cause(err) != ErrServiceExist {
		t.Fatal("expect(error)", ErrServiceExist, "got", errors.Cause(err))
	}

	err = r.Prepare("test")
	if err != nil {
		t.Fatal(err)
	}

	service, exist := r.routes["test"]
	if !exist {
		t.Fatal("expect exist got !exist")
	}

	if service.openFn != nil {
		t.Fatal("expect openFn nil")
	}

	if service.closeFn != nil {
		t.Fatal("expect closeFn nil")
	}

	err = r.Prepare("test")
	if err == nil {
		t.Fatal("expect(error)", ErrServiceExist, "got", err)
	}

}

func TestRouter_Add(t *testing.T) {
	r := NewRouter()

	func() {
		defer func() {
			if r := recover(); r != nil {
				return
			}
			t.Fatal("expect panic")
		}()

		r.Add("test", nil, func() error {
			return nil
		})
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				return
			}
			t.Fatal("expect panic")
		}()

		r.Add("test", func() (net.Conn, error) {
			return nil, nil
		}, nil)
	}()

	ok := r.Add("test", func() (net.Conn, error) {
		return nil, nil
	}, func() error {
		return nil
	})

	if ok {
		t.Fatal("expect !ok got ok")
	}

	err := r.Prepare("test")
	if err != nil {
		t.Fatal(err)
	}

	ok = r.Add("test", func() (net.Conn, error) {
		return nil, nil
	}, func() error {
		return nil
	})

	if !ok {
		t.Fatal("expect ok got !ok")
	}
}

func TestRouter_Get(t *testing.T) {
	r := NewRouter()
	service := r.Get("test")
	if service != nil {
		t.Fatal("expect nil got", service)
	}

	err := r.Prepare("test")
	if err != nil {
		t.Fatal(err)
	}

	service = r.Get("test")
	if service == nil {
		t.Fatal("expect not nil")
	}
}

func TestRouter_Remove(t *testing.T) {
	r := NewRouter()
	err := r.Prepare("test")
	if err != nil {
		t.Fatal(err)
	}

	_, exist := r.routes["test"]
	if !exist {
		t.Fatal("expect exist got !exist")
	}

	r.Remove("test")
	_, exist = r.routes["test"]
	if exist {
		t.Fatal("expect !exist got exist")
	}

	err = r.Prepare("test")
	if err != nil {
		t.Fatal(err)
	}

	closeCalled := false
	ok := r.Add("test", func() (net.Conn, error) {
		return nil, nil
	}, func() error {
		closeCalled = true
		return nil
	})

	if !ok {
		t.Fatal("expect ok got !ok")
	}

	r.Remove("test")
	if !closeCalled {
		t.Fatal("expect closeCalled got !closeCalled")
	}

	r.Remove("")
}

func TestRouter_All(t *testing.T) {
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}

	func() {
		r := NewRouter()
		ss := r.All()
		if len(ss) != 0 {
			t.Fatal(len(ss), "want", 0)
		}
	}()

	func() {
		r := NewRouter()
		must(r.Prepare("abc"))
		must(r.Prepare("dec"))
		must(r.Prepare("23445"))
		must(r.Prepare("test"))

		ss := r.All()

		if len(ss) != 4 {
			t.Fatal(len(ss), "want", 4)
		}

		expectNames := []string{"23445", "abc", "dec", "test"}
		for i, s := range ss {
			if s.Name() != expectNames[i] {
				t.Fatal(s.Name(), "want", expectNames[i])
			}
		}

	}()
}
