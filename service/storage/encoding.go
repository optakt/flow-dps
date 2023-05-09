package storage

import (
	"encoding/binary"
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

func EncodeKey(prefix uint8, segments ...interface{}) []byte {
	key := []byte{prefix}
	var val []byte
	for _, segment := range segments {
		switch s := segment.(type) {
		case uint64:
			val = make([]byte, 8)
			binary.BigEndian.PutUint64(val, s)
		case flow.Identifier:
			val = make([]byte, 32)
			copy(val, s[:])
		case ledger.Path:
			val = make([]byte, 32)
			copy(val, s[:])
		case flow.StateCommitment:
			val = make([]byte, 32)
			copy(val, s[:])
		default:
			panic(fmt.Sprintf("unknown type (%T)", segment))
		}
		key = append(key, val...)
	}

	return key
}
