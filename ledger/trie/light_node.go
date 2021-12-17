// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

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

const (
	legacyVersion   = uint16(0)
	encodingVersion = uint16(1)
)

// LightNode is a node that is formatted in a way that it can be easily encoded and written on disk,
// as part of a checkpoint. Instead of having pointers to its children, it stores
// that information using the index at which its children are in the light node index.
type LightNode struct {
	// Positions of the left and right children in the index.
	LIndex uint64
	RIndex uint64

	// Path and hash of the node in the trie.
	Path      []byte
	HashValue []byte

	// Height where the node is at.
	Height uint16
	// Height at which the node skips if it is an extension.
	Skip uint16
}

// IndexMap is a map used to index light nodes. It keeps track of the position of
// each node, and is also used to avoid issues with duplicate nodes.
type IndexMap map[Node]uint64

// ToLightNode converts the given node into a light node and indexes its position in the given index.
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

	// By calling node.Hash we ensure that the node hash is computed and does not get stored in a dirty state.
	hash := node.Hash()

	// Set the common node data.
	lightNode := LightNode{
		LIndex:    leftIndex,
		RIndex:    rightIndex,
		Height:    node.Height(),
		HashValue: hash[:],
	}

	// Add the missing data that is specific to each different node type.
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

// FromLightNode transforms a light node into a proper node.
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

// EncodeLightNode encodes a light node into a slice of bytes.
func EncodeLightNode(lightNode *LightNode, store dps.Store) []byte {

	var payload *ledger.Payload
	if len(lightNode.Path) > 0 && lightNode.Skip == 0 {
		// Since the node has a path and no skip value, we know it is a leaf node.
		h, err := hash.ToHash(lightNode.HashValue)
		if err != nil {
			panic(fmt.Errorf("fatal error: invalid hash in node: %w", err))
		}

		// Retrieve the payload from the store
		payload, err = store.Retrieve(h)
		if err != nil {
			panic(fmt.Errorf("fatal error: missing payload from store: %w", err))
		}
	}
	encPayload := encoding.EncodePayload(payload)

	// Length is calculated using:
	// 	* encoding version (2 bytes)
	// 	* LIndex (8 bytes)
	// 	* RIndex (8 bytes)
	// 	* Height (2 bytes)
	// 	* Skip (2 bytes)
	// 	* Length of path (32 bytes)
	// 	* Length of hash (32 bytes)
	//	* Length of encoded payload (variable)
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

// DecodeLightNode reads encoded light node data and returns a light node.
// It supports the legacy encoding format from FlowGo as well as the new optimized format.
func DecodeLightNode(reader io.Reader, store dps.Store) (*LightNode, error) {

	// Length is calculated using:
	// 	* encoding version (2 bytes)
	var buf [2]byte
	_, err := io.ReadFull(reader, buf[:])
	if err != nil {
		return nil, fmt.Errorf("could not read light node encoding version: %w", err)
	}

	version, _, err := utils.ReadUint16(buf[:])
	if err != nil {
		return nil, fmt.Errorf("could not read light node: %w", err)
	}

	switch version {
	case encodingVersion:
		return decodeNewFormat(reader, store)
	case legacyVersion:
		return decodeLegacyFormat(reader, store)
	default:
		return nil, fmt.Errorf("unsupported encoding version: %d", version)
	}
}

// Decodes a newly-formatted light node.
func decodeLegacyFormat(reader io.Reader, store dps.Store) (*LightNode, error) {

	// Length is calculated using:
	// 	* LIndex (8 bytes)
	// 	* RIndex (8 bytes)
	// 	* Height (2 bytes)
	// 	* Skip (2 bytes)
	length := 8 + 8 + 2 + 2
	buf := make([]byte, length)
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("could not read fixed-length part of light node: %w", err)
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
	if payload != nil {
		store.Save(h, payload)
	}

	return &lightNode, nil
}

// Decodes a legacy-formatted light node.
func decodeNewFormat(reader io.Reader, store dps.Store) (*LightNode, error) {
	// Length is calculated using:
	// 	* Height (2 bytes)
	// 	* LIndex (8 bytes)
	// 	* RIndex (8 bytes)
	// 	* MaxDepth (2 bytes) — Ignored
	// 	* RegisterCount (8 bytes) — Ignored
	buf := make([]byte, 2+8+8+2+8)
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("could not read fixed-length part of light node: %w", err)
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
	if payload != nil {
		store.Save(h, payload)
	}

	return &lightNode, nil
}
