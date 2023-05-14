package convert

import (
	"fmt"

	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/encoding/rlp"
	"github.com/onflow/flow-go/model/flow"
)

// KeyToRegisterID converts a ledger key into a register ID.
func KeyToRegisterID(key ledger.Key) (flow.RegisterID, error) {
	if len(key.KeyParts) != 2 ||
		key.KeyParts[0].Type != state.KeyPartOwner ||
		key.KeyParts[1].Type != state.KeyPartKey {
		return flow.RegisterID{}, fmt.Errorf("key not in expected format: %s", key.String())
	}

	return flow.RegisterID{
		Owner: string(key.KeyParts[0].Value),
		Key:   string(key.KeyParts[1].Value),
	}, nil
}

// RegistersToBytes converts a slice of ledger registers into a slice of byte slices.
func RegistersToBytes(values flow.RegisterIDs) [][]byte {
	bb := make([][]byte, 0, len(values))
	for _, value := range values {
		bb = append(bb, value.Bytes())
	}
	return bb

}

// BytesToRegisters converts a slice of byte slices into a slice of ledger registers.
func BytesToRegisters(bb [][]byte) (flow.RegisterIDs, error) {
	values := make(flow.RegisterIDs, 0, len(bb))
	unmarshaler := rlp.NewMarshaler()
	for _, b := range bb {
		var decoded flow.RegisterID
		err := unmarshaler.Unmarshal(b, &decoded)
		if err != nil {
			return nil, fmt.Errorf("could not decode register ID: %w", err)
		}

		values = append(values, decoded)
	}
	return values, nil
}
