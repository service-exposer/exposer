package exposer

import "encoding/json"

type HandshakeIncoming struct {
	Command string          `json:"cmd"`
	Details json.RawMessage `json:"details"`
}

type HandshakeOutgoing struct {
	Command string      `json:"cmd"`
	Details interface{} `json:"details"`
}
