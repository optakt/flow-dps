package rest

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/onflow/flow-go/ledger"
)

func EncodeKey(key ledger.Key) string {
	partsEncoded := make([]string, 0, len(key.KeyParts))
	for _, part := range key.KeyParts {
		partEncoded := fmt.Sprintf("%d.%x", part.Type, part.Value)
		partsEncoded = append(partsEncoded, partEncoded)
	}
	return strings.Join(partsEncoded, ",")
}

func EncodeKeys(keys []ledger.Key) string {
	keysEncoded := make([]string, 0, len(keys))
	for _, key := range keys {
		keyEncoded := EncodeKey(key)
		keysEncoded = append(keysEncoded, keyEncoded)
	}
	return strings.Join(keysEncoded, ":")
}

func DecodeKey(keyEncoded string) (ledger.Key, error) {
	var key ledger.Key
	partsEncoded := strings.Split(keyEncoded, ",")
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

func DecodeKeys(keysParam string) ([]ledger.Key, error) {
	if len(keysParam) == 0 {
		return nil, fmt.Errorf("keys parameter is empty")
	}
	keysEncoded := strings.Split(keysParam, ":")
	var keys []ledger.Key
	for _, keyEncoded := range keysEncoded {
		key, err := DecodeKey(keyEncoded)
		if err != nil {
			return nil, fmt.Errorf("could not decode key: %w (key: %s)", err, keyEncoded)
		}
		keys = append(keys, key)
	}
	return keys, nil
}
