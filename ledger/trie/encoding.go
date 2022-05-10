package trie

const (
	encNodeCountSize = 8
	encTrieCountSize = 2
)

func encodeNode(node Node, lIndex uint64, rIndex uint64, scratch []byte) []byte {
	switch n := node.(type) {
	case *Branch:
		return encodeBranch(n, lIndex, rIndex, scratch)

	case *Extension:
		return encodeExtension(n, lIndex, scratch)

	case *Leaf:
		return encodeLeaf(n, scratch)

	case nil:
		panic("unexpected nil node when encoding checkpoint")
	}

	return nil
}

func encodeTrie(t *Trie, rootIndex uint64, scratch []byte) []byte {
	return nil // FIXME: Implement
}

// encodeLeafNode encodes leaf node in the following format:
// - node type (1 byte)
// - height (2 bytes)
// - hash (32 bytes)
// - path (32 bytes)
// - payload (4 bytes + n bytes)
// Encoded leaf node size is 81 bytes (assuming length of hash/path is 32 bytes) +
// length of encoded payload size.
// Scratch buffer is used to avoid allocs. It should be used directly instead
// of using append.  This function uses len(scratch) and ignores cap(scratch),
// so any extra capacity will not be utilized.
// WARNING: The returned buffer is likely to share the same underlying array as
// the scratch buffer. Caller is responsible for copying or using returned buffer
// before scratch buffer is used again.
func encodeLeaf(node Node, scratch []byte) []byte {
	return nil // FIXME: Implement
}

func encodeExtension(node Node, childIndex uint64, scratch []byte) []byte {
	return nil // FIXME: Implement
}

func encodeBranch(node Node, lIndex uint64, rIndex uint64, scratch []byte) []byte {
	return nil // FIXME: Implement
}
