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

	"golang.org/x/sync/semaphore"

	"github.com/onflow/flow-go/ledger"

	"github.com/onflow/flow-go/ledger/common/encoding"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/ledger/common/utils"
)

const (
	encodingVersion = uint16(0)
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

	Payload []byte
	Path    []byte

	// Height at which the node skips if it is an extension.
	Count uint8
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
		if !n.clean {
			return nil, fmt.Errorf("cannot convert dirty extension to light node")
		}
	case *Branch:
		if !n.clean {
			return nil, fmt.Errorf("cannot convert dirty branch to light node")
		}
	}

	h := node.Hash(semaphore.NewWeighted(1), 0)
	lightNode := LightNode{
		HashValue: h[:],
	}

	switch n := node.(type) {
	case *Branch:
		leftIndex, found := index[n.left]
		if !found {
			h := n.left.Hash(semaphore.NewWeighted(1), 0)
			return nil, fmt.Errorf("missing node with hash %s", hex.EncodeToString(h[:]))
		}
		rightIndex, found := index[n.right]
		if !found {
			h := n.right.Hash(semaphore.NewWeighted(1), 0)
			return nil, fmt.Errorf("missing node with hash %s", hex.EncodeToString(h[:]))
		}
		lightNode.LIndex = leftIndex
		lightNode.RIndex = rightIndex

	case *Extension:
		lightNode.Count = n.count
		childIndex, found := index[n.child]
		if !found {
			h := n.child.Hash(semaphore.NewWeighted(1), 0)
			return nil, fmt.Errorf("missing node with hash %s", hex.EncodeToString(h[:]))
		}
		lightNode.LIndex = childIndex
		lightNode.Path = n.path[:]

	case *Leaf:
		lightNode.Payload = encoding.EncodePayload(n.payload)
		lightNode.Path = n.path[:]
	}

	return &lightNode, nil
}

// FromLightNode transforms a light node into a proper node.
func FromLightNode(ln *LightNode, nodes []Node) (Node, error) {
	hash, err := hash.ToHash(ln.HashValue)
	if err != nil {
		return nil, fmt.Errorf("invalid hash in light node: %w", err)
	}

	// Branch node.
	if ln.LIndex != 0 && ln.RIndex != 0 {
		println("Branch")
		// Since it does not have a path, this node is a branch.
		return &Branch{
			left:  nodes[ln.LIndex],
			right: nodes[ln.RIndex],
			hash:  hash,
			clean: true,
		}, nil
	}

	println(ln.Path)
	println(ln.Payload)
	println(ln.HashValue)

	path, err := ledger.ToPath(ln.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path in light node: %w", err)
	}

	// Extension node.
	if ln.LIndex != 0 {
		// Since it only has a single child, this node is an extension.
		return &Extension{
			child: nodes[ln.LIndex],
			count: ln.Count,
			path:  &path,
			hash:  hash,
			clean: true,
		}, nil
	}

	// Since it has no children, this node is a leaf.
	payload, err := encoding.DecodePayload(ln.Payload)
	if err != nil {
		return nil, fmt.Errorf("invalid payload in light node: %w", err)
	}

	return &Leaf{
		path:    &path,
		payload: payload,
		hash:    hash,
		clean:   true,
	}, nil
}

// DecodeLightNode reads encoded light node data and returns a light node.
// It supports the legacy encoding format from FlowGo as well as the new optimized format.
func DecodeLightNode(reader io.Reader) (*LightNode, error) {

	// Length is calculated using:
	// 	* encoding version (2 bytes)
	buf := make([]byte, 2)
	read, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("could not read light node encoding version: %w", err)
	}
	if read != len(buf) {
		return nil, fmt.Errorf("not enough bytes in encoding version: %d expected %d", read, len(buf))
	}

	version, _, err := utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not read light node: %w", err)
	}

	switch version {
	case encodingVersion:
		return decodeLegacyFormat(reader)
	default:
		return nil, fmt.Errorf("unsupported encoding version: %d", version)
	}
}

// Decodes a newly-formatted light node.
func decodeLegacyFormat(reader io.Reader) (*LightNode, error) {

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
	read, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("could not read fixed-length part of light node: %w", err)
	}
	if read != len(buf) {
		return nil, fmt.Errorf("not enough bytes read: %d expected %d", read, len(buf))
	}

	// Read height but ignore it.
	_, buf, err = utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node height: %w", err)
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
	// Read and ignore Max Depth value.
	_, buf, err = utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node max depth: %w", err)
	}
	// Read and ignore Register Count value.
	_, _, err = utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node register count: %w", err)
	}

	// Read path.
	lightNode.Path, err = utils.ReadShortDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node path: %w", err)
	}

	// Read payload.
	encPayload, err := utils.ReadLongDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read light node payload: %w", err)
	}
	payload, err := encoding.DecodePayload(encPayload)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node payload: %w", err)
	}
	lightNode.Payload = encoding.EncodePayload(payload)

	// Read hash.
	lightNode.HashValue, err = utils.ReadShortDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node hash: %w", err)
	}

	return &lightNode, nil
}
