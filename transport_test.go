package exposer

import (
	"io"
	"net"
	"os"
	"testing"
)

func TestTransport_Reply(t *testing.T) {
	s, c := net.Pipe()
	trans := NewTransport(c)
	if trans.isHandshakeDone != false {
		t.Fatal(trans.isHandshakeDone)
	}

	go func() {
		defer trans.conn.Close()
		err := trans.Reply("test", nil)
		if err != nil {
			t.Fatal(err)
		}
	}()

	io.Copy(os.Stdout, s)

}
