// Copyright 2021 Alvalor S.A.
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

package dps

import (
	"github.com/onflow/flow-go/model/flow"
)

var Networks map[flow.ChainID]Network

type Network struct {
	FungibleToken    flow.Address
	FlowFees         flow.Address
	StakingTable     flow.Address
	LockedTokens     flow.Address
	StakingProxy     flow.Address
	NonFungibleToken flow.Address
	Tokens           map[flow.Address]Token
}

type Token struct {
	Symbol   string
	Vault    string
	Receiver string
}

func init() {

	// Hard-code the Flow token storage paths from here:
	// https://github.com/onflow/flow-core-contracts/blob/master/contracts/FlowToken.cdc
	flowToken := Token{
		Symbol:   "FLOW",
		Vault:    "/storage/flowTokenVault",
		Receiver: "/public/flowTokenReceiver",
	}

	// Hard-code test network parameters from:
	// https://docs.onflow.org/core-contracts
	testnet := Network{
		FungibleToken:    flow.HexToAddress("9a0766d93b6608b7"),
		FlowFees:         flow.HexToAddress("912d5440f7e3769e"),
		StakingTable:     flow.HexToAddress("9eca2b38b18b5dfe"),
		LockedTokens:     flow.HexToAddress("95e019a17d0e23d7"),
		StakingProxy:     flow.HexToAddress("7aad92e5a0715d21"),
		NonFungibleToken: flow.HexToAddress("631e88ae7f1d7c20"),
		Tokens: map[flow.Address]Token{
			flow.HexToAddress("7e60df042a9c0868"): flowToken,
		},
	}
	Networks[flow.Testnet] = testnet

	// Hard-code main network parameters from:
	// https://docs.onflow.org/core-contracts
	mainnet := Network{
		FungibleToken:    flow.HexToAddress("f233dcee88fe0abe"),
		FlowFees:         flow.HexToAddress("f919ee77447b7497"),
		StakingTable:     flow.HexToAddress("8624b52f9ddcd04a"),
		LockedTokens:     flow.HexToAddress("8d0e87b65159ae63"),
		StakingProxy:     flow.HexToAddress("62430cf28c26d095"),
		NonFungibleToken: flow.HexToAddress("1d7e57aa55817448"),
		Tokens: map[flow.Address]Token{
			flow.HexToAddress("1654653399040a61"): flowToken,
		},
	}
	Networks[flow.Mainnet] = mainnet
}
