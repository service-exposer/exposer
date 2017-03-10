package keepalive

import (
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/service-exposer/exposer"
)

var (
	DefaultInterval = 20 * time.Second
	DefaultTimeout  = 30 * time.Second
)

var (
	ErrTimeout = errors.New("timeout")
)

const (
	CMD_PING = "ping"
	CMD_PONG = "pong"
)

const (
	EVENT_TIMEOUT = "event:timeout"
)

func ServerSide(timeout time.Duration) exposer.HandshakeHandleFunc {
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	var (
		mutex        = new(sync.Mutex)
		lastPingTime = time.Now()
	)

	var once = new(sync.Once)

	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		once.Do(func() {
			go func() {
				for range time.Tick(timeout) {
					var done = false
					mutex.Lock()
					if time.Now().Sub(lastPingTime) > timeout {
						proto.Emit(EVENT_TIMEOUT, nil)
						done = true
					}
					mutex.Unlock()

					if done {
						return
					}
				}
			}()
		})

		switch cmd {
		case CMD_PING:
			mutex.Lock()
			lastPingTime = time.Now()
			mutex.Unlock()

			return proto.Reply(CMD_PONG, nil)
		case EVENT_TIMEOUT:
			return errors.Trace(ErrTimeout)
		}

		return errors.New("unknow cmd: " + cmd)
	}
}

func ClientSide(timeout, interval time.Duration) exposer.HandshakeHandleFunc {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	if interval == 0 {
		interval = DefaultInterval
	}

	if interval >= timeout {
		panic("interval MUST less than timeout")
	}

	var (
		mutex        = new(sync.Mutex)
		lastPingTime = time.Now()
	)

	var once = new(sync.Once)

	return func(proto *exposer.Protocal, cmd string, details []byte) error {
		once.Do(func() {
			go func() {
				for range time.Tick(timeout) {
					var done = false
					mutex.Lock()
					if time.Now().Sub(lastPingTime) > timeout {
						proto.Emit(EVENT_TIMEOUT, nil)
						done = true
					}
					mutex.Unlock()

					if done {
						return
					}
				}
			}()
		})

		switch cmd {
		case CMD_PONG:
			mutex.Lock()
			lastPingTime = time.Now()
			mutex.Unlock()

			time.Sleep(interval)
			return proto.Reply(CMD_PING, nil)
		case EVENT_TIMEOUT:
			return errors.Trace(ErrTimeout)
		}

		return errors.New("unknow cmd: " + cmd)

	}
}
