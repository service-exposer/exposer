package exposer

import (
	"io"
	"net"
	"os"
	"testing"
)

func TestProtocal_Reply(t *testing.T) {
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

	io.Copy(os.Stdout, s)

}
