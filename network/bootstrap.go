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

package network

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go/follower"
)

type Bootstrap struct {
	Identities []Identity `json:"Identities"`
}

type Identity struct {
	NodeID        string `json:"NodeID"`
	Address       string `json:"Address"`
	Role          string `json:"Role"`
	Stake         int    `json:"Stake"`
	StakingPubKey string `json:"StakingPubKey"`
	NetworkPubKey string `json:"NetworkPubKey"`
}

func RetrieveIdentities(path string) ([]follower.BootstrapNodeInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not read bootstrap directory: %w", err)
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("could not read root protocol state snapshot: %w", err)
	}

	var info Bootstrap
	err = json.Unmarshal(b, &info)
	if err != nil {
		return nil, fmt.Errorf("could not decode root protocol state snapshot: %w", err)
	}

	var nodeInfo []follower.BootstrapNodeInfo
	for _, id := range info.Identities {
		host, portStr, err := net.SplitHostPort(id.Address)
		if err != nil {
			return nil, fmt.Errorf("could not parse node address: %w", err)
		}

		port, err := strconv.ParseUint(portStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse node port: %w", err)
		}

		// FIXME: How do we know the algorithm the key uses?
		key, err := crypto.DecodePublicKeyHex(crypto.ECDSA_P256, id.NetworkPubKey)
		if err != nil {
			return nil, fmt.Errorf("could not parse node address: %w", err)
		}

		nodeInfo = append(nodeInfo, follower.BootstrapNodeInfo{
			Host:             host,
			Port:             uint(port),
			NetworkPublicKey: key,
		})
	}

	return nodeInfo, nil
}
