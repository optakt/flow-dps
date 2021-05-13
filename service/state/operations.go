package state

import (
	"encoding/binary"
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/awfm9/flow-dps/model"
)

// Retrieve gets any arbitrary value from a given key.
func Retrieve(key []byte, value *[]byte) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return fmt.Errorf("unable to retrieve value: %w", err)
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("unable to copy value: %w", err)
		}

		*value = val

		return nil
	}
}

func RetrieveLastHeight(height *uint64) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		item, err := txn.Get(Encode(model.PrefixLastHeight))
		if err != nil {
			return fmt.Errorf("unable to retrieve last height: %w", err)
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("unable to copy last height: %w", err)
		}
		*height = binary.BigEndian.Uint64(val)

		return nil
	}
}

func SetLastHeight(height uint64) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		err := txn.Set(Encode(model.PrefixLastHeight), Encode(height))
		if err != nil {
			return fmt.Errorf("unable to persist last height: %w", err)
		}

		return nil
	}
}

func RetrieveLastCommit(commit *[]byte) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		item, err := txn.Get(Encode(model.PrefixLastCommit))
		if err != nil {
			return fmt.Errorf("unable to retrieve last commit: %w", err)
		}

		_, err = item.ValueCopy(*commit)
		if err != nil {
			return fmt.Errorf("unable to copy last commit: %w", err)
		}

		return nil
	}
}

func SetLastCommit(commit []byte) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		err := txn.Set(Encode(model.PrefixLastCommit), commit)
		if err != nil {
			return fmt.Errorf("unable to persist last commit: %w", err)
		}

		return nil
	}
}
