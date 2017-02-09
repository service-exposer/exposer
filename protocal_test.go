package exposer

import (
	"encoding/json"
	"net"
	"reflect"
	"testing"
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

}
