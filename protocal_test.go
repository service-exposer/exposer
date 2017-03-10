package exposer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/inconshreveable/muxado"
	"github.com/juju/errors"
)

func TestProtocal_Reply(t *testing.T) {
	{
		s, c := net.Pipe()
		proto := NewProtocal(c)
		if proto.isHandshakeDone != false {
			t.Fatal(proto.isHandshakeDone)
		}

		go func() {
			defer proto.conn.Close()
			err := proto.Reply("test", nil)
			if err != nil {
				t.Fatal(err)
			}
		}()

		var reply HandshakeIncoming
		err := json.NewDecoder(s).Decode(&reply)
		if err != nil {
			t.Fatal(err)
		}

		var replyExpect = HandshakeIncoming{
			Command: "test",
			Details: json.RawMessage([]byte("null")),
		}

		if !reflect.DeepEqual(&replyExpect, &reply) {
			t.Fatal("expect", replyExpect, "got", reply)
		}
	}

	{
		_, c := net.Pipe()
		proto := NewProtocal(c)
		proto.isHandshakeDone = true

		func() {
			defer proto.conn.Close()
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("expect panic")
				}
			}()

			proto.Reply("test", nil)
		}()
	}

	{
		_, c := net.Pipe()
		proto := NewProtocal(c)
		c.Close()

		err := proto.Reply("test", nil)
		if err == nil {
			t.Fatal("expect err")
		}
	}
}

func TestProtocal_Forword(t *testing.T) {
	func() { // normal
		c, c1 := net.Pipe()
		c2, c3 := net.Pipe()
		defer c.Close()
		defer c2.Close()

		proto_c := NewProtocal(c)

		//defer c1.Close()

		go proto_c.Forward(c2)

		c1.Write([]byte("test"))
		c1.Close()

		data, err := ioutil.ReadAll(c3)
		if err != nil {
			t.Fatal(err)
		}

		if string(data) != "test" {
			t.Fatal("expect", "test", "got", string(data))
		}
	}()

	func() { // design usage
		c, c1 := net.Pipe()
		c2, c3 := net.Pipe()
		defer c.Close()

		proto := NewProtocal(c)
		proto.On = func(proto *Protocal, cmd string, details []byte) error {
			switch cmd {
			case "forward":
				proto.Forward(c2)
			}
			return nil
		}

		go proto.Handle()

		c1.Write([]byte(`{"cmd":"forward","details":null}test`))
		c1.Close()

		data, err := ioutil.ReadAll(c3)
		if err != nil {
			t.Fatal(err)
		}

		if string(data) != "test" {
			t.Fatal("expect", "test", "got", string(data))
		}
	}()
}

func Test_newReadWriteCloser(t *testing.T) {
	s, c := net.Pipe()
	defer s.Close()

	go func() {
		s.Write([]byte("world"))
		s.Close()
	}()

	buf := &bytes.Buffer{}
	buf.WriteString("hello")

	rwc := newReadWriteCloser(buf, c)

	data, err := ioutil.ReadAll(rwc)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "helloworld" {
		t.Error("expect", "helloworld", "got", string(data))
	}
}

func TestProtocal_Multiplex(t *testing.T) {
	func() { // test lib github.com/inconshreveable/muxado
		s, c := net.Pipe()
		defer s.Close()

		session_s := muxado.Server(s, nil)

		defer session_s.Close()

		go func() {
			for {
				conn, err := session_s.Accept()
				if err != nil {
					return
				}

				go func(conn net.Conn) { // echo service
					defer conn.Close()

					io.Copy(conn, conn)
				}(conn)
			}
		}()

		session := muxado.Client(c, nil)
		defer session.Close()

		n := 100
		wg := &sync.WaitGroup{}
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(i int) {
				defer wg.Done()

				conn, err := session.Open()
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()

				wbuf := &bytes.Buffer{}

				conn.Write(wbuf.Bytes())

				rbuf := make([]byte, wbuf.Len())

				_, err = io.ReadAtLeast(conn, rbuf, len(rbuf))
				if err != nil {
					t.Fatal(err, string(rbuf))
				}

				if !bytes.Equal(rbuf, wbuf.Bytes()) {
					t.Fatal("expect", string(wbuf.Bytes()), "got", string(rbuf))
				}
			}(i)
		}

		wg.Wait()
	}()

	func() {
		s, c := net.Pipe()
		defer s.Close()

		session_s := NewProtocal(s).Multiplex(false)

		defer session_s.Close()

		go func() {
			for {
				conn, err := session_s.Accept()
				if err != nil {
					return
				}

				go func(conn net.Conn) { // echo service
					defer conn.Close()

					io.Copy(conn, conn)
				}(conn)
			}
		}()

		session := NewProtocal(c).Multiplex(true)
		defer session.Close()

		n := 100
		wg := &sync.WaitGroup{}
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(i int) {
				defer wg.Done()

				conn, err := session.Open()
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()

				wbuf := &bytes.Buffer{}
				fmt.Fprint(wbuf, "hello:", i)

				conn.Write(wbuf.Bytes())

				rbuf := make([]byte, wbuf.Len())

				_, err = io.ReadAtLeast(conn, rbuf, len(rbuf))
				if err != nil {
					t.Fatal(err, string(rbuf))
				}

				if !bytes.Equal(rbuf, wbuf.Bytes()) {
					t.Fatal("expect", string(wbuf.Bytes()), "got", string(rbuf))
				}
			}(i)
		}

		wg.Wait()
	}()

	func() {
		s, c := net.Pipe()
		defer s.Close()

		proto_s := NewProtocal(s)
		proto_s.On = func(proto *Protocal, cmd string, details []byte) error {
			switch cmd {
			case "multiplex":
				err := proto.Reply("multiplex:reply", nil)
				if err != nil {
					t.Fatal(err)
				}

				session := proto.Multiplex(false)
				defer session.Close()

				for {
					conn, err := session.Accept()
					if err != nil {
						return errors.Trace(err)
					}

					go func(conn net.Conn) { // echo service
						defer conn.Close()

						io.Copy(conn, conn)
					}(conn)
				}
			}

			return nil
		}
		go proto_s.Handle()

		proto_c := NewProtocal(c)
		proto_c.On = func(proto *Protocal, cmd string, details []byte) error {
			switch cmd {
			case "multiplex:reply":
				session := proto.Multiplex(true)
				defer session.Close()

				n := 100
				wg := &sync.WaitGroup{}
				wg.Add(n)
				for i := 0; i < n; i++ {
					go func(i int) {
						defer wg.Done()

						conn, err := session.Open()
						if err != nil {
							t.Fatal(err)
						}
						defer conn.Close()

						wbuf := &bytes.Buffer{}

						conn.Write(wbuf.Bytes())

						rbuf := make([]byte, wbuf.Len())

						_, err = io.ReadAtLeast(conn, rbuf, len(rbuf))
						if err != nil {
							t.Fatal(err, string(rbuf))
						}

						if !bytes.Equal(rbuf, wbuf.Bytes()) {
							t.Fatal("expect", string(wbuf.Bytes()), "got", string(rbuf))
						}

					}(i)
				}

				wg.Wait()
			}

			return nil
		}

		proto_c.Request("multiplex", nil)
	}()
}

func TestProtocal_Emit(t *testing.T) {
	func() {
		c, _ := net.Pipe()
		proto := NewProtocal(c)
		const (
			EVENT_TIMEOUT = "event:timeout"
		)

		waitTimeout := make(chan bool)
		proto.On = func(proto *Protocal, cmd string, details []byte) error {
			switch cmd {
			case EVENT_TIMEOUT:
				waitTimeout <- true
			}
			return errors.New("unknow cmd:" + cmd)
		}
		go proto.Handle()

		err := proto.Emit(EVENT_TIMEOUT, nil)
		if err != nil {
			t.Fatal(err)
		}

		select {
		case <-waitTimeout:
		case <-time.After(time.Millisecond * 100):
			t.Fatal("expect EVENT_TIMEOUT")
		}

	}()
}

func TestProtocal_Wait(t *testing.T) {
	var (
		ErrTest = errors.New("test error")
	)

	func() {
		c, _ := net.Pipe()
		proto := NewProtocal(c)
		const (
			EVENT_ERROR = "event:error"
		)

		proto.On = func(proto *Protocal, cmd string, details []byte) error {
			switch cmd {
			case EVENT_ERROR:
				return ErrTest
			}
			return errors.New("unknow cmd:" + cmd)
		}
		go proto.Handle()

		err := proto.Emit(EVENT_ERROR, nil)
		if err != nil {
			t.Fatal(err)
		}

		err = proto.Wait()
		if errors.Cause(err) != ErrTest {
			t.Fatal(err)
		}
	}()

	func() {
		s, c := net.Pipe()

		proto_server := NewProtocal(s)
		proto_server.On = func(proto *Protocal, cmd string, details []byte) error {
			return ErrTest
		}
		go proto_server.Handle()

		proto_client := NewProtocal(c)
		proto_client.On = func(proto *Protocal, cmd string, details []byte) error {
			return nil
		}
		go proto_client.Request("", nil)

		err := proto_server.Wait()
		if errors.Cause(err) != ErrTest {
			t.Fatal(errors.Cause(err), "want", ErrTest)
		}

		err = proto_client.Wait()
		if errors.Cause(err) == nil {
			t.Fatal(errors.Cause(err), "want", nil)
		}
	}()
}

func TestNewProtocalWithParent(t *testing.T) {
	conn, _ := net.Pipe()
	func() {
		proto := NewProtocalWithParent(nil, conn)
		if proto.parent != nil {
			t.Fatal(proto.parent, "expect nil")
		}
	}()

	func() {
		parent_proto := NewProtocal(conn)
		proto := NewProtocalWithParent(parent_proto, conn)
		if proto.parent != parent_proto {
			t.Fatal(proto.parent, "expect", parent_proto)
		}
	}()
}
