package generator

import (
	"time"
)

// DictionaryKind represents the resources that the dictionary is trained to compress efficiently.
type DictionaryKind string

// Supported dictionary kinds.
const (
	KindPayloads     DictionaryKind = "payloads"
	KindEvents       DictionaryKind = "events"
	KindTransactions DictionaryKind = "transactions"
)

func (k DictionaryKind) String() string {
	return string(k)
}

type dictionary struct {
	kind DictionaryKind
	raw  []byte
	size int

	ratio    float64
	duration time.Duration
}
