package payload

import (
	"encoding/binary"

	"github.com/onflow/flow-go/model/flow"
)

// lookupKey is the encoded format of the storage key for looking up register value
type lookupKey struct {
	encoded []byte
}

func newLookupKey(height uint64, reg flow.RegisterID) *lookupKey {
	key := lookupKey{
		encoded: make([]byte, 0, len(reg.Owner)+1+len(reg.Key)+1+heightSuffixLen),
	}

	key.encoded = append(key.encoded, []byte(reg.Owner)...)
	key.encoded = append(key.encoded, 0x00)
	key.encoded = append(key.encoded, []byte(reg.Key)...)
	key.encoded = append(key.encoded, 0x00)

	// Encode the height getting it to 1s compliment and switching byte order.
	// (Prefix iteration in pebble does not support reverse iteration.)
	onesCompliment := ^height
	key.encoded = binary.BigEndian.AppendUint64(key.encoded, onesCompliment)

	return &key
}

// Bytes returns the encoded lookup key.
func (h lookupKey) Bytes() []byte {
	return h.encoded
}
