package trie

import (
	"encoding/binary"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/ledger/common/utils"
	"github.com/optakt/flow-dps/codec/zbor"
)

const (
	nodeTypeLeaf uint8 = iota + 1
	nodeTypeBranch
	nodeTypeExtension
)

const (
	encNodeTypeSize      = 1
	encHashSize          = hash.HashLen
	encPathSize          = ledger.PathLen
	encPayloadLengthSize = 4
	encNodeCountSize     = 8
	encTrieCountSize     = 2
	encExtCountSize      = 1
	encNodeIndexSize     = 8

	encodedTrieSize = encNodeIndexSize + encHashSize

	payloadEncodingVersion = 1
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

	default:
		panic("unexpected node type when encoding checkpoint")
	}
}

func encodeTrie(t *Trie, rootIndex uint64, scratch []byte) []byte {
	buf := scratch
	if len(scratch) < encodedTrieSize {
		buf = make([]byte, encodedTrieSize)
	}

	pos := 0

	// Encode root node index (8 bytes Big Endian)
	binary.BigEndian.PutUint64(buf, rootIndex)
	pos += encNodeIndexSize

	// Encode hash (32-bytes hashValue)
	rootHash := t.RootHash()
	copy(buf[pos:], rootHash[:])
	pos += encHashSize

	return buf[:pos]
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
func encodeLeaf(leaf *Leaf, scratch []byte) []byte {
	encPayloadSize := encodedPayloadLength(leaf.payload, payloadEncodingVersion)

	encodedNodeSize := encNodeTypeSize +
		encHashSize +
		encPathSize +
		encPayloadLengthSize +
		encPayloadSize

	// buf uses received scratch buffer if it's large enough.
	// Otherwise, a new buffer is allocated.
	// buf is used directly so len(buf) must not be 0.
	// buf will be re-sliced to proper size before being returned from this function.
	buf := scratch
	if len(scratch) < encodedNodeSize {
		buf = make([]byte, encodedNodeSize)
	}

	pos := 0

	// Encode node type (1 byte)
	buf[pos] = nodeTypeLeaf
	pos += encNodeTypeSize

	// Encode hash (32 bytes)
	copy(buf[pos:], leaf.hash[:])
	pos += encHashSize

	// Encode path (32 bytes)
	copy(buf[pos:], leaf.path[:])
	pos += encHashSize

	binary.BigEndian.PutUint32(buf[pos:], uint32(encPayloadSize))
	pos += encPayloadLengthSize

	encPayload, err := zbor.NewCodec().Marshal(leaf.payload)
	if err != nil {
		panic(err)
	}

	buf = utils.AppendLongData(buf, encPayload)

	return buf
}

func encodeExtension(ext *Extension, childIndex uint64, scratch []byte) []byte {

	encodedNodeSize := encNodeTypeSize +
		encHashSize +
		encPathSize +
		encExtCountSize +
		encNodeIndexSize

	// buf uses received scratch buffer if it's large enough.
	// Otherwise, a new buffer is allocated.
	// buf is used directly so len(buf) must not be 0.
	// buf will be re-sliced to proper size before being returned from this function.
	buf := scratch
	if len(scratch) < encodedNodeSize {
		buf = make([]byte, encodedNodeSize)
	}

	pos := 0

	// Encode node type (1 byte)
	buf[pos] = nodeTypeExtension
	pos += encNodeTypeSize

	// Encode hash (32 bytes)
	copy(buf[pos:], ext.hash[:])
	pos += encHashSize

	// Encode path (32 bytes)
	copy(buf[pos:], ext.path[:])
	pos += encHashSize

	// Encode extension count (1 byte)
	buf[pos] = ext.count
	pos += encExtCountSize

	// Encode child index (8 bytes Big Endian)
	binary.BigEndian.PutUint64(buf[pos:], childIndex)
	pos += encNodeIndexSize

	return buf
}

func encodeBranch(branch *Branch, lIndex uint64, rIndex uint64, scratch []byte) []byte {
	encodedNodeSize := encNodeTypeSize +
		encHashSize +
		encNodeIndexSize +
		encNodeIndexSize

	// buf uses received scratch buffer if it's large enough.
	// Otherwise, a new buffer is allocated.
	// buf is used directly so len(buf) must not be 0.
	// buf will be re-sliced to proper size before being returned from this function.
	buf := scratch
	if len(scratch) < encodedNodeSize {
		buf = make([]byte, encodedNodeSize)
	}

	pos := 0

	// Encode node type (1 byte)
	buf[pos] = nodeTypeBranch
	pos += encNodeTypeSize

	// Encode hash (32 bytes)
	copy(buf[pos:], branch.hash[:])
	pos += encHashSize

	// Encode left child index (8 bytes Big Endian)
	binary.BigEndian.PutUint64(buf[pos:], lIndex)
	pos += encNodeIndexSize

	// Encode right child index (8 bytes Big Endian)
	binary.BigEndian.PutUint64(buf[pos:], rIndex)
	pos += encNodeIndexSize

	return scratch
}

func encodedPayloadLength(p *ledger.Payload, version uint16) int {
	if p == nil {
		return 0
	}
	switch version {
	case 0:
		// In version 0, payload is encoded as:
		//   encode key length (4 bytes) + encoded key +
		//   encoded value length (8 bytes) + encode value
		return 4 + encodedKeyLength(&p.Key, version) + 8 + len(p.Value)
	default:
		// In version 1 and later, payload is encoded as:
		//   encode key length (4 bytes) + encoded key +
		//   encoded value length (4 bytes) + encode value
		return 4 + encodedKeyLength(&p.Key, version) + 4 + len(p.Value)
	}
}

func encodedKeyLength(k *ledger.Key, version uint16) int {
	// Key is encoded as: number of key parts (2 bytes) and for each key part,
	// the key part size (4 bytes) + encoded key part (n bytes).
	size := 2 + 4*len(k.KeyParts)
	for _, kp := range k.KeyParts {
		size += encodedKeyPartLength(&kp, version)
	}
	return size
}

func encodedKeyPartLength(kp *ledger.KeyPart, _ uint16) int {
	// Key part is encoded as: type (2 bytes) + value
	return 2 + len(kp.Value)
}
