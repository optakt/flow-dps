package archive

import (
	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
)

// DefaultOptions returns the default Badger options preferred by the DPS for its index database.
func DefaultOptions(dir string) badger.Options {
	return badger.DefaultOptions(dir).
		WithMaxTableSize(256 << 20).
		WithValueLogFileSize(64 << 20).
		WithTableLoadingMode(options.FileIO).
		WithValueLogLoadingMode(options.FileIO).
		WithNumMemtables(1).
		WithKeepL0InMemory(false).
		WithCompactL0OnClose(false).
		WithNumLevelZeroTables(1).
		WithNumLevelZeroTablesStall(2).
		WithLoadBloomsOnOpen(false).
		WithIndexCacheSize(2000 << 20).
		WithBlockCacheSize(0).
		WithLogger(nil)
}
