package storage

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/go-multierror"
)

// Fallback goes through the provided operations until one of them succeeds.
// If all of them fail, a multi-error with all errors is returned.
func Fallback(ops ...func(*badger.Txn) error) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		var errs error
		for _, op := range ops {
			err := op(tx)
			if err == nil {
				return nil
			}

			errs = multierror.Append(errs, err)
		}

		return errs
	}
}

// Combine goes through the provided operations until one of them fails.
// When the first one fails, the related error is returned.
func Combine(ops ...func(*badger.Txn) error) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		for _, op := range ops {
			err := op(tx)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func (l *Library) retrieve(key []byte, v interface{}) func(tx *badger.Txn) error {
	return func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return fmt.Errorf("could not get value (key: %x): %w", key, err)
		}

		err = item.Value(func(val []byte) error {
			return l.codec.Unmarshal(val, v)
		})
		if err != nil {
			return fmt.Errorf("could not decode value (key: %x): %w", key, err)
		}

		return nil
	}
}

func (l *Library) save(key []byte, value interface{}) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		val, err := l.codec.Marshal(value)
		if err != nil {
			return fmt.Errorf("could not encode value (key: %x): %w", key, err)
		}

		err = tx.Set(key, val)
		if err != nil {
			return fmt.Errorf("could not set value (key: %x): %w", key, err)
		}

		return nil
	}
}

func (l *Library) batchWrite(key []byte, value interface{}) func(*badger.WriteBatch) error {
	return func(writeBatch *badger.WriteBatch) error {
		val, err := l.codec.Marshal(value)
		if err != nil {
			return fmt.Errorf("could not encode value (key: %x): %w", key, err)
		}

		err = writeBatch.Set(key, val)
		if err != nil {
			return fmt.Errorf("could not set value (key: %x): %w", key, err)
		}

		return nil
	}
}
