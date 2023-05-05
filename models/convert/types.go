package convert

import (
	"time"

	"github.com/onflow/flow-go/model/flow"
)

// TypesToStrings converts a slice of flow event types into a slice of strings.
func TypesToStrings(types []flow.EventType) []string {
	ss := make([]string, 0, len(types))
	for _, typ := range types {
		ss = append(ss, string(typ))
	}
	return ss
}

// StringsToTypes converts a slice of strings into a slice of flow event types.
func StringsToTypes(ss []string) []flow.EventType {
	types := make([]flow.EventType, 0, len(ss))
	for _, s := range ss {
		types = append(types, flow.EventType(s))
	}
	return types
}

// RosettaTime converts a time into a Rosetta-compatible timestamp.
func RosettaTime(t time.Time) int64 {
	return t.UnixNano() / 1_000_000
}
