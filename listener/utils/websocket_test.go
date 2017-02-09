package utils

import (
	"io"
	"sync"
	"testing"
)

func TestWebsocket(t *testing.T) {
	ln, err := WebsocketListener("tcp", "localhost:9775")
	if err != nil {
		t.Fatal(err)
	}

	closeWg := &sync.WaitGroup{}
	closeWg.Add(1)
	go func() {
		defer closeWg.Done()
		defer ln.Close()

		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}

			go func() {
				defer conn.Close()

				io.Copy(conn, conn)
			}()
		}
	}()

	n := 100
	wg := &sync.WaitGroup{}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			conn, err := DialWebsocket("ws://localhost:9775/randomstring/it-is-no-matter/")
			if err != nil {
				t.Fatal(err)
			}

			conn.Write([]byte("hello"))
			buf := make([]byte, 5)

			_, err = io.ReadAtLeast(conn, buf, 5)
			if err != nil {
				t.Fatal(err)
			}

			if string(buf) != "hello" {
				t.Fatal("expect", "hello", "got", string(buf))
			}

		}()
	}
	wg.Wait()
	ln.Close()

	closeWg.Wait()
}
