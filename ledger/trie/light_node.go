package trie

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/encoding"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/ledger/common/utils"
	"github.com/optakt/flow-dps/models/dps"
)

type LightNode struct {
	LIndex uint64
	RIndex uint64

	Path      []byte
	HashValue []byte

	// Height where the node is at.
	Height uint16
	// Height at which the node skips if it is an extension.
	Skip uint16
}

type IndexMap map[Node]uint64

func ToLightNode(node Node, index IndexMap) (*LightNode, error) {
	leftIndex, found := index[node.LeftChild()]
	if !found {
		hash := node.LeftChild().Hash()
		return nil, fmt.Errorf("missing node with hash %s", hex.EncodeToString(hash[:]))
	}
	rightIndex, found := index[node.RightChild()]
	if !found {
		hash := node.RightChild().Hash()
		return nil, fmt.Errorf("missing node with hash %s", hex.EncodeToString(hash[:]))
	}

	// By calling node.Hash we ensure that the node hash is computed and will not be stored dirty.
	hash := node.Hash()
	lightNode := LightNode{
		LIndex:    leftIndex,
		RIndex:    rightIndex,
		Height:    node.Height(),
		HashValue: hash[:],
	}

	switch n := node.(type) {
	case *Extension:
		lightNode.Skip = n.skip
		lightNode.Path = n.path[:]

	case *Leaf:
		lightNode.Path = n.path[:]

	case *Branch:
		// No extra data is needed in the light node.
	}

	return &lightNode, nil
}

func FromLightNode(ln *LightNode, nodes []Node) (Node, error) {
	hash, err := hash.ToHash(ln.HashValue)
	if err != nil {
		return nil, fmt.Errorf("invalid hash in light node: %w", err)
	}

	if len(ln.Path) == 0 {
		// Since it does not have a path, this node is a branch.
		return &Branch{
			lChild: nodes[ln.LIndex],
			rChild: nodes[ln.RIndex],
			height: ln.Height,
			hash:   hash,
			dirty:  false,
		}, nil
	}

	path, err := ledger.ToPath(ln.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path in light node: %w", err)
	}

	if ln.Skip > 0 {
		// Since it has a skip value, this node is an extension.
		return &Extension{
			lChild: nodes[ln.LIndex],
			rChild: nodes[ln.RIndex],
			height: ln.Height,
			path:   path,
			skip:   ln.Skip,
			hash:   hash,
			dirty:  false,
		}, nil
	}

	// Since it has a path and has no skip value, this node is a leaf.
	return &Leaf{
		path:   path,
		hash:   hash,
		height: ln.Height,
	}, nil
}

const (
	legacyVersion   = uint16(0)
	encodingVersion = uint16(1)
)

func EncodeLightNode(lightNode *LightNode, store dps.Store) []byte {

	var payload *ledger.Payload
	if len(lightNode.Path) > 0 && lightNode.Skip == 0 {
		// Since the node has a path and no skip value, we know it is a leaf node.
		h, err := hash.ToHash(lightNode.HashValue)
		if err != nil {
			panic(fmt.Errorf("fatal error: invalid hash in node: %w", err))
		}

		payload, err = store.Retrieve(h)
		if err != nil {
			panic(fmt.Errorf("fatal error: missing payload from store: %w", err))
		}
	}
	encPayload := encoding.EncodePayload(payload)

	length := 2 + 8 + 8 + 2 + 2 + len(lightNode.Path) + len(lightNode.HashValue) + len(encPayload)
	buf := make([]byte, length)

	buf = utils.AppendUint16(buf, encodingVersion)
	buf = utils.AppendUint64(buf, lightNode.LIndex)
	buf = utils.AppendUint64(buf, lightNode.RIndex)
	buf = utils.AppendUint16(buf, lightNode.Height)
	buf = utils.AppendUint16(buf, lightNode.Skip)
	buf = utils.AppendShortData(buf, lightNode.Path)
	buf = utils.AppendShortData(buf, lightNode.HashValue)
	buf = utils.AppendLongData(buf, encPayload)

	return buf
}

func DecodeLightNode(reader io.Reader, store dps.Store) (*LightNode, error) {
	var buf [2]byte
	read, err := io.ReadFull(reader, buf[:])
	if err != nil {
		return nil, fmt.Errorf("could not read light node encoding version: %w", err)
	}
	if read != len(buf) {
		return nil, fmt.Errorf("not enough bytes read (got %d, expected %d)", read, len(buf))
	}

	version, _, err := utils.ReadUint16(buf[:])
	if err != nil {
		return nil, fmt.Errorf("could not read light node: %w", err)
	}

	switch version {
	case encodingVersion:
		return decodeLightNode(reader, store)
	case legacyVersion:
		return decodeLegacyNode(reader, store)
	default:
		return nil, fmt.Errorf("unsupported encoding version: expected %d got %d", encodingVersion, version)
	}
}

func decodeLightNode(reader io.Reader, store dps.Store) (*LightNode, error) {
	length := 8 + 8 + 2 + 2
	buf := make([]byte, length)
	read, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("could not read fixed-length part of light node: %w", err)
	}
	if read != len(buf) {
		return nil, fmt.Errorf("not enough bytes read (got %d, expected %d)", read, len(buf))
	}

	var lightNode LightNode
	lightNode.LIndex, buf, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node left index: %w", err)
	}
	lightNode.RIndex, buf, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node right index: %w", err)
	}
	lightNode.Height, buf, err = utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node height: %w", err)
	}
	lightNode.Skip, buf, err = utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node skipped height: %w", err)
	}

	lightNode.Path, err = utils.ReadShortDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node path: %w", err)
	}
	lightNode.HashValue, err = utils.ReadShortDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node hash: %w", err)
	}

	encPayload, err := utils.ReadLongDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read light node payload: %w", err)
	}
	payload, err := encoding.DecodePayload(encPayload)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node payload: %w", err)
	}
	// We need to store the decoded payload in the store so that if new node insertions come up,
	// the store can be looked up to recompute node hashes as they are moved to new heights.
	h, err := hash.ToHash(lightNode.HashValue)
	if err != nil {
		return nil, fmt.Errorf("invalid hash in light node: %w", err)
	}
	store.Save(h, payload)

	return &lightNode, nil
}

// Legacy nodes have the following attributes and sizes:
// version        - 2B -> already parsed in parent call
// height         - 2B
// left index     - 8B
// right index    - 8B
// max depth      - 2B -> not used
// register count - 8B -> not used
// path           - 2B + Path Length
// payload        - 4B + Payload Length -> not used
// hash           - 2B + Hash Length
func decodeLegacyNode(reader io.Reader, store dps.Store) (*LightNode, error) {
	buf := make([]byte, 2+8+8+2+8)
	read, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("could not read fixed-length part of light node: %w", err)
	}
	if read != len(buf) {
		return nil, fmt.Errorf("not enough bytes read (got %d, expected %d)", read, len(buf))
	}

	var lightNode LightNode
	lightNode.Height, buf, err = utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node height: %w", err)
	}
	lightNode.LIndex, buf, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node left index: %w", err)
	}
	lightNode.RIndex, buf, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node right index: %w", err)
	}
	// Ignore the max depth value.
	_, buf, err = utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node max depth: %w", err)
	}
	// Ignore the register count value.
	_, buf, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node register count: %w", err)
	}

	lightNode.Path, err = utils.ReadShortDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node path: %w", err)
	}
	encPayload, err := utils.ReadLongDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read light node payload: %w", err)
	}
	payload, err := encoding.DecodePayload(encPayload)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node payload: %w", err)
	}
	lightNode.HashValue, err = utils.ReadShortDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node hash: %w", err)
	}

	// We need to store the decoded payload in the store so that if new node insertions come up,
	// the store can be looked up to recompute node hashes as they are moved to new heights.
	h, err := hash.ToHash(lightNode.HashValue)
	if err != nil {
		return nil, fmt.Errorf("invalid hash in light node: %w", err)
	}
	store.Save(h, payload)

	return &lightNode, nil
}
