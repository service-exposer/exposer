package forward

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/service-exposer/exposer"
	"github.com/service-exposer/exposer/listener/utils"
)

func TestForward(t *testing.T) {
	authFn := func(k string) bool {
		return true
	}
	authFn("")

	const (
		MESSAGE = "hello world"
	)
	var (
		remote_addr     = "127.0.0.2:9210"
		forward_ws_addr = "127.0.0.2:9211"
		local_addr      = "127.0.0.2:9212"
	)

	// remote server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(MESSAGE))
	})

	remote_ln, err := net.Listen("tcp", remote_addr)
	if err != nil {
		t.Fatal(err)
	}
	defer remote_ln.Close()

	go http.Serve(remote_ln, nil)

	time.Sleep(time.Second)
	log.Print("remote server")

	// forward server
	forward_ws_ln, err := utils.WebsocketListener("tcp", forward_ws_addr)
	//forward_ws_ln, err := net.Listen("tcp", forward_ws_addr)
	if err != nil {
		t.Fatal(err)
	}
	defer forward_ws_ln.Close()

	go exposer.Serve(forward_ws_ln, func(conn net.Conn) exposer.ProtocalHandler {
		proto := exposer.NewProtocal(conn)
		proto.On = ServerSide(authFn)
		return proto
	})
	time.Sleep(time.Second)
	log.Print("forward server")

	// local listen
	local_ln, err := net.Listen("tcp", local_addr)
	if err != nil {
		t.Fatal(err)
	}
	defer local_ln.Close()

	conn, err := utils.DialWebsocket("ws://" + forward_ws_addr)
	//conn, err := net.Dial("tcp", forward_ws_addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	proto := exposer.NewProtocal(conn)
	proto.On = ClientSide(Forward{
		Network: "tcp",
		Address: remote_addr,
	}, local_ln)

	go proto.Request(CMD_AUTH, &Auth{
		Key: "",
	})

	time.Sleep(1 * time.Second)

	// access remote server by local address
	resp, err := http.Get("http://" + local_addr)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if string(data) != MESSAGE {
		t.Fatal("expect", MESSAGE, "got", string(data))
	}
}
