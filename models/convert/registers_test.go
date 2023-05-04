package convert_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-archive/models/convert"
	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

const (
	expectedOwner = "verylongownerstring123456789012345678901234567890"
	expectedKey   = "key"
)

// Test_KeyToRegisterID_Invalid tests the KeyToRegisterID function using
// table tests for a set of invalid inputs.
func Test_KeyToRegisterID_Invalid(t *testing.T) {
	t.Parallel()

	invalidKeys := []ledger.Key{
		{},
		{KeyParts: []ledger.KeyPart{{}}},
		{KeyParts: []ledger.KeyPart{{Type: state.KeyPartOwner}}},
		{KeyParts: []ledger.KeyPart{{Type: state.KeyPartOwner, Value: []byte("owner")}}},
		{KeyParts: []ledger.KeyPart{{Type: state.KeyPartOwner, Value: []byte("owner")}, {}}},
		{KeyParts: []ledger.KeyPart{{Type: state.KeyPartOwner, Value: []byte("owner")}, {Type: 99}}},
		{KeyParts: []ledger.KeyPart{{Type: state.KeyPartOwner, Value: []byte("owner")}, {Type: state.KeyPartKey, Value: []byte("key")}, {}}},
	}

	for _, invalidKey := range invalidKeys {
		reg, err := convert.KeyToRegisterID(invalidKey)
		require.Error(t, err)
		require.Equal(t, flow.RegisterID{}, reg)
	}
}

// Test_KeyToRegisterID_Valid tests the KeyToRegisterID function a valid input.
func Test_KeyToRegisterID_Valid(t *testing.T) {
	t.Parallel()

	ledgerKey, err := convert.KeyToRegisterID(
		ledger.Key{KeyParts: []ledger.KeyPart{
			{Type: state.KeyPartOwner, Value: []byte(expectedOwner)},
			{Type: state.KeyPartKey, Value: []byte(expectedKey)},
		}},
	)
	require.NoError(t, err)
	require.Equal(t, expectedOwner, ledgerKey.Owner)
	require.Equal(t, expectedKey, ledgerKey.Key)
}

// Test_RegistersToBytes_RoundTrip tests the RegistersToBytes and BytesToRegisters round trip.
func Test_RegistersToBytes_RoundTrip(t *testing.T) {
	t.Parallel()

	ledgerKeys := flow.RegisterIDs{
		flow.RegisterID{Owner: expectedOwner, Key: expectedKey},
		flow.RegisterID{Owner: "owner", Key: "key"},
	}

	bb := convert.RegistersToBytes(ledgerKeys)
	require.Len(t, bb, 2)

	decoded, err := convert.BytesToRegisters(bb)
	require.NoError(t, err)
	require.Len(t, decoded, 2)

	require.ElementsMatch(t, ledgerKeys, decoded)
}
