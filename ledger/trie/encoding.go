package trie

import (
	"encoding/binary"
	"io"

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

func Decode(reader io.Reader) (*Trie, error) {
	return nil, nil
}

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

func decodeNode(data []byte, allNodes []Node) (Node, []byte) {
	nodeType := data[0]
	data = data[1:]

	switch nodeType {
	case nodeTypeBranch:
		return decodeBranch(data, allNodes)

	case nodeTypeExtension:
		return decodeExtension(data, allNodes)

	case nodeTypeLeaf:
		return decodeLeaf(data)

	default:
		panic("unexpected node type when decoding checkpoint")
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

func decodeTrie(data []byte) (uint64, hash.Hash, []byte) {
	pos := 0

	// Decode root node index (8 bytes Big Endian)
	rootIndex := binary.BigEndian.Uint64(data)
	pos += encNodeIndexSize

	// Decode hash (32-bytes hashValue)
	var rootHash hash.Hash
	copy(rootHash[:], data[pos:pos+encHashSize])
	pos += encHashSize

	return rootIndex, rootHash, data[pos:]
}

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

func decodeLeaf(data []byte) (*Leaf, []byte) {
	pos := 0

	// Decode hash (32 bytes)
	hash := hash.Hash{}
	copy(hash[:], data[pos:])
	pos += encHashSize

	// Decode path (32 bytes)
	path := ledger.Path{}
	copy(path[:], data[pos:])
	pos += encHashSize

	// Decode payload length (4 bytes)
	payloadLength := binary.BigEndian.Uint32(data[pos:])
	pos += encPayloadLengthSize

	// Decode payload
	payloadBytes := data[pos : pos+int(payloadLength)]
	var payload ledger.Payload
	err := zbor.NewCodec().Unmarshal(payloadBytes, &payload)
	if err != nil {
		panic(err) // FIXME
	}
	pos += int(payloadLength)

	node := Leaf{
		hash:    hash,
		path:    &path,
		payload: &payload,
	}

	return &node, data[pos:]
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

func decodeExtension(data []byte, nodes []Node) (*Extension, []byte) {
	pos := 0

	// Decode hash (32 bytes)
	hash := hash.Hash{}
	copy(hash[:], data[pos:])
	pos += encHashSize

	// Decode path (32 bytes)
	path := ledger.Path{}
	copy(path[:], data[pos:])
	pos += encHashSize

	// Decode extension count (1 byte)
	count := data[pos]
	pos += encExtCountSize

	// Decode child index (8 bytes Big Endian)
	childIndex := binary.BigEndian.Uint64(data[pos:])
	pos += encNodeIndexSize

	node := Extension{
		hash:  hash,
		path:  &path,
		count: count,
		child: nodes[childIndex],
	}

	return &node, data[pos:]
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

func decodeBranch(data []byte, nodes []Node) (*Branch, []byte) {
	pos := 0

	// Decode hash (32 bytes)
	hash := hash.Hash{}
	copy(hash[:], data[pos:])
	pos += encHashSize

	// Decode left child index (8 bytes Big Endian)
	lIndex := binary.BigEndian.Uint64(data[pos:])
	pos += encNodeIndexSize

	// Decode right child index (8 bytes Big Endian)
	rIndex := binary.BigEndian.Uint64(data[pos:])
	pos += encNodeIndexSize

	node := Branch{
		hash:  hash,
		left:  nodes[lIndex],
		right: nodes[rIndex],
	}

	return &node, data[pos:]
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
