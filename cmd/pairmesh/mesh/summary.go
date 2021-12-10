package mesh

import (
	"fmt"
	"time"
)

type State byte

const (
	StatePending State = 0
	StateRelay   State = 1
	StateP2P     State = 2
)

var stateStringify = [...]string{
	StatePending: "pending",
	StateRelay:   "relay",
	StateP2P:     "p2p",
}

// String implements the fmt.Stringer interface
func (s State) String() string {
	if s > StateP2P {
		return fmt.Sprintf("unknown(%d)", s)
	}
	return stateStringify[s]
}

type (
	Device struct {
		Name   string `json:"name"`
		IPv4   string `json:"ipv4"`
		Status State  `json:"status"`
	}

	Network struct {
		ID      uint64   `json:"id"`
		Name    string   `json:"name"`
		Devices []Device `json:"devices"`
	}

	Summary struct {
		LastChangedAt time.Time `json:"-"`
		MyDevices     []Device  `json:"my_devices"`
		Networks      []Network `json:"networks"`
	}
)
