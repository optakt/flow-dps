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
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/onflow/flow-go/ledger/common/utils"
)

type LightTrie struct {
	RootIndex uint64
	RootHash  []byte
}

func ToLightTrie(t *Trie, index IndexMap) (*LightTrie, error) {
	rootIndex, found := index[t.RootNode()]
	if !found {
		hash := t.RootHash()
		return nil, fmt.Errorf("missing node with hash %s", hex.EncodeToString(hash[:]))
	}

	hash := t.RootHash()
	lt := LightTrie{
		RootIndex: rootIndex,
		RootHash:  hash[:],
	}

	return &lt, nil
}

// FIXME
func FromLightTrie(lt *LightTrie, nodes []Node) (*Trie, error) {
	t := NewTrie(nodes[lt.RootIndex])
	rootHash := t.RootHash()
	if !bytes.Equal(lt.RootHash, rootHash[:]) {
		return nil, fmt.Errorf("could not restore trie: roothash does not match")
	}
	return t, nil
}

func EncodeLightTrie(lightTrie *LightTrie) []byte {
	length := 2 + 8 + len(lightTrie.RootHash)
	buf := make([]byte, length)

	buf = utils.AppendUint16(buf, encodingVersion)
	buf = utils.AppendUint64(buf, lightTrie.RootIndex)
	buf = utils.AppendShortData(buf, lightTrie.RootHash)

	return buf
}

func DecodeLightTrie(reader io.Reader) (*LightTrie, error) {
	var lightTrie LightTrie

	buf := make([]byte, 2)
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("could not read light trie decoding version: %w", err)
	}

	version, _, err := utils.ReadUint16(buf)
	if err != nil {
		return nil, fmt.Errorf("could not read light trie: %w", err)
	}

	if version > encodingVersion {
		return nil, fmt.Errorf("unsupported version %d > %d", version, encodingVersion)
	}

	// read root uint64 RootIndex
	buf = make([]byte, 8)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("could not read fixed-length part of light trie: %w", err)
	}

	rootIndex, _, err := utils.ReadUint64(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node root index: %w", err)
	}
	lightTrie.RootIndex = rootIndex

	rootHash, err := utils.ReadShortDataFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not decode light node root hash: %w", err)
	}
	lightTrie.RootHash = rootHash

	return &lightTrie, nil
}
