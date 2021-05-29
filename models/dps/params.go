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

const (
	FlowBlockchain = "flow"
	FlowMainnet    = flow.Mainnet
	FlowTestnet    = flow.Testnet
	FlowSymbol     = "FLOW"
	FlowDecimals   = 8
)

var FlowParams = make(map[flow.ChainID]Params)

type Params struct {
	ChainID          flow.ChainID
	FungibleToken    flow.Address
	FlowFees         flow.Address
	StakingTable     flow.Address
	LockedTokens     flow.Address
	StakingProxy     flow.Address
	NonFungibleToken flow.Address
	Tokens           map[string]Token
}

type Token struct {
	Symbol   string
	Address  flow.Address
	Type     string
	Vault    string
	Receiver string
	Balance  string
}

func init() {

	// Hard-code the Flow token storage paths from here:
	// https://github.com/onflow/flow-core-contracts/blob/master/contracts/FlowToken.cdc
	flowToken := Token{
		Symbol:   FlowSymbol,
		Address:  flow.EmptyAddress,
		Type:     "FlowToken",
		Vault:    "/storage/flowTokenVault",
		Receiver: "/public/flowTokenReceiver",
		Balance:  "/public/flowTokenBalance",
	}

	// Hard-code test network parameters from:
	// https://docs.onflow.org/core-contracts
	flowToken.Address = flow.HexToAddress("7e60df042a9c0868")
	testnet := Params{
		ChainID:          FlowTestnet,
		FungibleToken:    flow.HexToAddress("9a0766d93b6608b7"),
		FlowFees:         flow.HexToAddress("912d5440f7e3769e"),
		StakingTable:     flow.HexToAddress("9eca2b38b18b5dfe"),
		LockedTokens:     flow.HexToAddress("95e019a17d0e23d7"),
		StakingProxy:     flow.HexToAddress("7aad92e5a0715d21"),
		NonFungibleToken: flow.HexToAddress("631e88ae7f1d7c20"),
		Tokens: map[string]Token{
			flowToken.Symbol: flowToken,
		},
	}
	FlowParams[testnet.ChainID] = testnet

	// Hard-code main network parameters from:
	// https://docs.onflow.org/core-contracts
	flowToken.Address = flow.HexToAddress("1654653399040a61")
	mainnet := Params{
		ChainID:          FlowMainnet,
		FungibleToken:    flow.HexToAddress("f233dcee88fe0abe"),
		FlowFees:         flow.HexToAddress("f919ee77447b7497"),
		StakingTable:     flow.HexToAddress("8624b52f9ddcd04a"),
		LockedTokens:     flow.HexToAddress("8d0e87b65159ae63"),
		StakingProxy:     flow.HexToAddress("62430cf28c26d095"),
		NonFungibleToken: flow.HexToAddress("1d7e57aa55817448"),
		Tokens: map[string]Token{
			flowToken.Symbol: flowToken,
		},
	}
	FlowParams[mainnet.ChainID] = mainnet
}
