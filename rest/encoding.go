package rest

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/onflow/flow-go/ledger"
)

func decodeKey(keyEncoded string) (ledger.Key, error) {

	// split into key parts
	partsEncoded := strings.Split(keyEncoded, ",")

	// split each part into type and value
	var key ledger.Key
	for _, partEncoded := range partsEncoded {
		tokens := strings.Split(partEncoded, ".")
		if len(tokens) != 2 {
			return ledger.Key{}, fmt.Errorf("every key part must be of format type.value (have: %s)", partEncoded)
		}
		typ, err := strconv.ParseUint(tokens[0], 10, 16)
		if err != nil {
			return ledger.Key{}, fmt.Errorf("could not parse key part type: %w", err)
		}
		val, err := hex.DecodeString(tokens[1])
		if err != nil {
			return ledger.Key{}, fmt.Errorf("could not decode key part value: %w", err)
		}
		part := ledger.KeyPart{
			Type:  uint16(typ),
			Value: val,
		}
		key.KeyParts = append(key.KeyParts, part)
	}

	return key, nil
}
