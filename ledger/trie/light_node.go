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

	// Hash of the node in the trie.
	HashValue []byte

	// Height at which the node skips if it is an extension.
	Skip uint8
}

// IndexMap is a map used to index light nodes. It keeps track of the position of
// each node, and is also used to avoid issues with duplicate nodes.
type IndexMap map[Node]uint64

// ToLightNode converts the given node into a light node and indexes its position in the given index.
func ToLightNode(node Node, index IndexMap) (*LightNode, error) {

	// We need to ensure that the nodes are not dirty before they can be converted
	// to light nodes, since we do not have access to their height here.
	switch n := node.(type) {
	case *Extension:
		if n.dirty {
			return nil, fmt.Errorf("cannot convert dirty extension to light node")
		}
	case *Branch:
		if n.dirty {
			return nil, fmt.Errorf("cannot convert dirty branch to light node")
		}
	}

	h := node.Hash(0, [32]byte{}, nil)
	lightNode := LightNode{
		HashValue: h[:],
	}

	switch n := node.(type) {
	case *Extension:
		lightNode.Skip = n.count

	case *Leaf:
		break // nothing to do.

	case *Branch:
		leftIndex, found := index[n.left]
		if !found {
			h := n.left.Hash(0, [32]byte{}, nil)
			return nil, fmt.Errorf("missing node with hash %s", hex.EncodeToString(h[:]))
		}
		rightIndex, found := index[n.right]
		if !found {
			h := n.right.Hash(0, [32]byte{}, nil)
			return nil, fmt.Errorf("missing node with hash %s", hex.EncodeToString(h[:]))
		}
		lightNode.LIndex = leftIndex
		lightNode.RIndex = rightIndex
	}

	return &lightNode, nil
}

// FromLightNode transforms a light node into a proper node.
func FromLightNode(ln *LightNode, nodes []Node) (Node, error) {
	hash, err := hash.ToHash(ln.HashValue)
	if err != nil {
		return nil, fmt.Errorf("invalid hash in light node: %w", err)
	}

	if ln.LIndex != 0 || ln.RIndex != 0 {
		// Since it does not have a path, this node is a branch.
		return &Branch{
			left:  nodes[ln.LIndex],
			right: nodes[ln.RIndex],
			hash:  hash,
			dirty: false,
		}, nil
	}

	if ln.Skip > 0 {
		// Since it has a skip value, this node is an extension.
		return &Extension{
			// FIXME: Handle child.
			count: ln.Skip, // FIXME: Rename skip
			hash:  hash,
			dirty: false,
		}, nil
	}

	// Since it has a path and has no skip value, this node is a leaf.
	return &Leaf{
		hash: hash,
	}, nil
}

// EncodeLightNode encodes a light node into a slice of bytes.
func EncodeLightNode(lightNode *LightNode, store dps.Store) []byte {

	var payload *ledger.Payload
	var err error
	if lightNode.Skip == 0 && lightNode.LIndex == 0 && lightNode.RIndex == 0 {
		// Since the node has a no children, we know it's a leaf node.
		var h hash.Hash
		h, err = hash.ToHash(lightNode.HashValue)
		if err != nil {
			panic(fmt.Errorf("fatal error: invalid hash in node: %w", err))
		}

		// Retrieve the payload from the store
		payload.Value, err = store.Retrieve(h)
		if err != nil {
			panic(fmt.Errorf("fatal error: missing payload from store: %w", err))
		}
	}

	encPayload := encoding.EncodePayload(payload)

	// Length is calculated using:
	// 	* Encoding version (2 bytes)
	// 	* LIndex (8 bytes)
	// 	* RIndex (8 bytes)
	// 	* Height (1 byte)
	// 	* Skip (1 byte) // FIXME: Rename?
	// 	* Length of path (32 bytes)
	// 	* Length of hash (32 bytes)
	//	* Length of encoded payload (variable)
	length := 2 + 8 + 8 + 1 + len(lightNode.HashValue) + len(encPayload)
	buf := make([]byte, length)

	buf = utils.AppendUint16(buf, encodingVersion)
	buf = utils.AppendUint64(buf, lightNode.LIndex)
	buf = utils.AppendUint64(buf, lightNode.RIndex)
	buf = utils.AppendUint8(buf, lightNode.Skip)
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
	// 	* Height (2 bytes)
	// 	* LIndex (8 bytes)
	// 	* RIndex (8 bytes)
	// 	* Max Depth (2 bytes) -> Skipped
	// 	* Register Count (8 bytes) -> Skipped
	//  * Path (32 bytes)
	//  * Payload (variable)
	//  * Hash (32 bytes)
	length := 2 + 8 + 8 + 2 + 8
	buf := make([]byte, length)
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("could not read fixed-length part of light node: %w", err)
	}

	// Read height but ignore it.
	_, buf, err = utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node height: %w", err)
	}

	// Subtract one height since the checkpoint has nodes with heights from
	// 256 to 1 instead of 255 to 0.
	var lightNode LightNode

	lightNode.LIndex, buf, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node left index: %w", err)
	}
	lightNode.RIndex, buf, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node right index: %w", err)
	}
	// Read and discard Max Depth value.
	_, buf, err = utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node max depth: %w", err)
	}
	// Read and discard Register Count value.
	_, _, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node register count: %w", err)
	}

	// Read path but ignore it.
	_, err = utils.ReadShortDataFromReader(reader)
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

	// We need to store the decoded payload in the store so that if new node insertions come up,
	// the store can be looked up to recompute node hashes as they are moved to new heights.
	lightNode.HashValue, err = utils.ReadShortDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node hash: %w", err)
	}
	h, err := hash.ToHash(lightNode.HashValue)
	if err != nil {
		return nil, fmt.Errorf("invalid hash in light node: %w", err)
	}
	if payload != nil {
		encoded := encoding.EncodePayload(payload)
		err = store.Save(h, encoded)
		if err != nil {
			return nil, fmt.Errorf("could not save light node payload: %w", err)
		}
	}

	return &lightNode, nil
}

// Decodes a legacy-formatted light node.
func decodeNewFormat(reader io.Reader, store dps.Store) (*LightNode, error) {
	// Length is calculated using:
	// 	* LIndex (8 bytes)
	// 	* RIndex (8 bytes)
	// 	* MaxDepth (1 byte) — Ignored
	// 	* RegisterCount (8 bytes) — Ignored
	buf := make([]byte, 8+8+2+8)
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
	// Ignore the max depth value.
	_, buf, err = utils.ReadUint8(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node max depth: %w", err)
	}
	// Ignore the register count value.
	_, buf, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node register count: %w", err)
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
		encoded := encoding.EncodePayload(payload)
		err = store.Save(h, encoded)
		if err != nil {
			return nil, fmt.Errorf("could not save light node payload: %w", err)
		}
	}

	return &lightNode, nil
}
