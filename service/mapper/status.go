package mapper

import "fmt"

// Status is a representation of the state machine's status.
type Status uint8

// The following is an enumeration of all possible statuses the
// state machine can have. The order in the enum reflects the order of normal transitions in the FSM
const (
	StatusInitialize Status = iota + 1
	StatusBootstrap
	StatusResume
	StatusUpdate
	StatusCollect
	StatusMap
	StatusIndex
	StatusForward
)

// String implements the Stringer interface. In order of
func (s Status) String() string {
	switch s {
	case StatusInitialize:
		return "initialize"
	case StatusBootstrap:
		return "bootstrap"
	case StatusResume:
		return "resume"
	case StatusUpdate:
		return "update"
	case StatusCollect:
		return "collect"
	case StatusMap:
		return "map"
	case StatusIndex:
		return "index"
	case StatusForward:
		return "forward"
	default:
		return fmt.Sprintf("invalid status %d", s)
	}
}
