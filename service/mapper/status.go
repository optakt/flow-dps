package mapper

import "fmt"

// Status is a representation of the state machine's status.
type Status uint8

// The following is an enumeration of all possible statuses the
// state machine can have.
const (
	StatusInitialize Status = iota + 1
	StatusBootstrap
	StatusResume
	StatusIndex
	StatusUpdate
	StatusCollect
	StatusMap
	StatusForward
)

// String implements the Stringer interface.
func (s Status) String() string {
	switch s {
	case StatusInitialize:
		return "initialize"
	case StatusBootstrap:
		return "bootstrap"
	case StatusResume:
		return "resume"
	case StatusIndex:
		return "index"
	case StatusUpdate:
		return "update"
	case StatusCollect:
		return "collect"
	case StatusMap:
		return "map"
	case StatusForward:
		return "forward"
	default:
		return fmt.Sprintf("invalid status %d", s)
	}
}
