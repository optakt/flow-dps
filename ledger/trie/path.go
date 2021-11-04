package trie

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
)

// CommonBits returns the number of matching bits within two paths.
func CommonBits(path1, path2 ledger.Path) int {
	for i := 0; i < ledger.NodeMaxHeight; i++ {
		if bitutils.Bit(path1[:], i) != bitutils.Bit(path2[:], i) {
			return i
		}
	}

	return ledger.NodeMaxHeight
}
