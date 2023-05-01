package payload

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/onflow/flow-go/model/flow"
	"github.com/stretchr/testify/require"
)

// Test_lookupKey_Bytes tests the lookup key encoding.
func Test_lookupKey_Bytes(t *testing.T) {
	t.Parallel()

	expectedHeight := uint64(777)
	key := newLookupKey(expectedHeight, flow.RegisterID{Owner: "owner", Key: "key"})

	// Test encoded Owner and Key
	require.Equal(t, []byte("owner\x00key\x00"), key.Bytes()[:10])

	// Test encoded height
	actualHeight := binary.BigEndian.Uint64(key.Bytes()[10:])
	require.Equal(t, math.MaxUint64-actualHeight, expectedHeight)

	// Test everything together
	require.Equal(t, []byte("owner\x00key\x00\xff\xff\xff\xff\xff\xff\xfc\xf6"), key.Bytes())
}
