package trie

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
)

// CommonBits returns the number of matching bits within two paths.
func CommonBits(path1, path2 ledger.Path) int {
	//fmt.Print("Comparing paths:\n1:\t")
	//for _, p1 := range path1 {
	//	fmt.Printf("%08b", p1)
	//}
	//fmt.Print("\n2:\t")
	//for _, p2 := range path2 {
	//	fmt.Printf("%08b", p2)
	//}
	//fmt.Println()
	for i := 0; i < ledger.NodeMaxHeight; i++ {
		if bitutils.Bit(path1[:], i) != bitutils.Bit(path2[:], i) {
			return i
		}
	}

	return ledger.NodeMaxHeight
}
