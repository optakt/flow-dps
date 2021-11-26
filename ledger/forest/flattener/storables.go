package flattener

type StorableNode struct {
	LIndex     uint64
	RIndex     uint64
	Height     uint16 // Height where the node is at
	Path       []byte // path
	HashValue  []byte

	// FIXME: Either get this from the DB when needed, or remove it entirely, since we don't need
	//  to restore payloads when we rebuild a trie as long as we have up to date hashes.
	//EncPayload []byte // encoded data for payload
}

// StorableTrie is a data structure for storing trie
type StorableTrie struct {
	RootIndex uint64
	RootHash  []byte
}
