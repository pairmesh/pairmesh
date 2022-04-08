package mesh

import (
	"fmt"
	"time"
)

// State represents the state of a mesh node
type State byte

// State is the state of a mesh node
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
	// Device is the struct of a device
	Device struct {
		Name   string `json:"name"`
		IPv4   string `json:"ipv4"`
		Status State  `json:"status"`
	}

	// Network is the struct of a network
	Network struct {
		ID      uint64   `json:"id"`
		Name    string   `json:"name"`
		Devices []Device `json:"devices"`
	}

	// Summary is the summary with last changed time, devices and networks
	Summary struct {
		LastChangedAt time.Time `json:"-"`
		MyDevices     []Device  `json:"my_devices"`
		Networks      []Network `json:"networks"`
	}
)
