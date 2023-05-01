package payload

import "github.com/cockroachdb/pebble"

// newMVCCComparer creates a new comparer with a
// custom Split function that separates the height from the rest of the key.
//
// This is needed for SeekPrefixGE to work.
func newMVCCComparer() *pebble.Comparer {
	comparer := *pebble.DefaultComparer
	comparer.Split = func(a []byte) int {
		return len(a) - heightSuffixLen
	}
	comparer.Name = "flow.MVCCComparer"

	return &comparer
}
