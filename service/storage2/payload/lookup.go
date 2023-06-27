package payload

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/onflow/flow-archive/service/storage2/config"
	"github.com/onflow/flow-go/model/flow"
)

// lookupKey is the encoded format of the storage key for looking up register value
type lookupKey struct {
	encoded []byte
}

func newLookupKey(height uint64, reg flow.RegisterID) *lookupKey {
	key := lookupKey{
		encoded: make([]byte, 0, len(reg.Owner)+1+len(reg.Key)+1+config.HeightSuffixLen),
	}

	// The lookup key used to find most recent value for a register.
	//
	// The "<owner>/<key>" part is the register key, which is used as a prefix to filter and iterate
	// through updated values at different heights, and find the most recent updated value at or below
	// a certain height.
	key.encoded = append(key.encoded, []byte(reg.Owner)...)
	key.encoded = append(key.encoded, '/')
	key.encoded = append(key.encoded, []byte(reg.Key)...)
	key.encoded = append(key.encoded, '/')

	// Encode the height getting it to 1s compliment (all bits flipped) and big-endian byte order.
	// (Prefix iteration in pebble does not support reverse iteration.)
	onesCompliment := ^height
	key.encoded = binary.BigEndian.AppendUint64(key.encoded, onesCompliment)

	return &key
}

func lookupKeyToRegisterID(lookupKey []byte) (uint64, flow.RegisterID, error) {
	// Find the first and second occurrence of '/' to split the lookup key.
	firstSlash := bytes.IndexByte(lookupKey, '/')
	if firstSlash == -1 {
		return 0, flow.RegisterID{}, fmt.Errorf("invalid lookup key format, can't find first slash")
	}

	secondSlash := bytes.IndexByte(lookupKey[firstSlash+1:], '/')
	if secondSlash == -1 {
		return 0, flow.RegisterID{}, fmt.Errorf("invalid lookup key format, can't find second slash")
	}

	// Extract owner, key, and height portions from the lookup key.
	owner := string(lookupKey[:firstSlash])
	key := string(lookupKey[firstSlash+1 : firstSlash+secondSlash+1])
	heightBytes := lookupKey[firstSlash+secondSlash+2:]

	// Decode the height from the remaining bytes in big-endian byte order.
	if len(heightBytes) != 8 {
		return 0, flow.RegisterID{}, fmt.Errorf("invalid lookup key format, invalid height bytes length (%v != 8)", len(heightBytes))
	}

	oneCompliment := binary.BigEndian.Uint64(heightBytes)
	height := ^oneCompliment
	reg := flow.RegisterID{Owner: owner, Key: key}

	return height, reg, nil
}

// Bytes returns the encoded lookup key.
func (h lookupKey) Bytes() []byte {
	return h.encoded
}
