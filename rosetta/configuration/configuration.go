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

package configuration

import (
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/meta"
)

const (
	blockchainUnknown = "network identifier has unknown blockchain field"
	networkUnknown    = "network identifier has unknown network field"
)

type Configuration struct {
	network    identifier.Network
	version    meta.Version
	statuses   []meta.StatusDefinition
	operations []string
	errors     []meta.ErrorDefinition
}

// New returns the configuration for a given Flow chain.
func New(chain flow.ChainID) *Configuration {

	network := identifier.Network{
		Blockchain: dps.FlowBlockchain,
		Network:    chain.String(),
	}

	version := meta.Version{
		RosettaVersion:    RosettaVersion,
		NodeVersion:       NodeVersion,
		MiddlewareVersion: MiddlewareVersion,
	}

	statuses := []meta.StatusDefinition{
		StatusCompleted,
	}

	operations := []string{
		OperationTransfer,
	}

	errors := []meta.ErrorDefinition{
		ErrorInternal,
		ErrorInvalidEncoding,
		ErrorInvalidFormat,
		ErrorInvalidNetwork,
		ErrorInvalidAccount,
		ErrorInvalidCurrency,
		ErrorInvalidBlock,
		ErrorInvalidTransaction,
		ErrorUnknownBlock,
		ErrorUnknownCurrency,
		ErrorUnknownTransaction,

		ErrorInvalidIntent,
		ErrorInvalidAuthorizers,
		ErrorInvalidPayer,
		ErrorInvalidProposer,
		ErrorInvalidScript,
		ErrorInvalidArguments,
		ErrorInvalidAmount,
		ErrorInvalidReceiver,
		ErrorInvalidSignature,
		ErrorInvalidKey,
		ErrorInvalidPayload,
		ErrorInvalidSignatures,
	}

	c := Configuration{
		network:    network,
		version:    version,
		statuses:   statuses,
		operations: operations,
		errors:     errors,
	}

	return &c
}

// Network returns the configuration's network identifier.
func (c *Configuration) Network() identifier.Network {
	return c.network
}

// Version returns the configuration's version information.
func (c *Configuration) Version() meta.Version {
	return c.version
}

// Statuses returns the configuration's status definitions.
func (c *Configuration) Statuses() []meta.StatusDefinition {
	return c.statuses
}

// Operations returns the configuration's supported operations.
func (c *Configuration) Operations() []string {
	return c.operations
}

// Errors returns the configuration's error definitions.
func (c *Configuration) Errors() []meta.ErrorDefinition {
	return c.errors
}

// Check verifies whether a network identifier matches with the configured one.
func (c *Configuration) Check(network identifier.Network) error {
	if network.Blockchain != c.network.Blockchain {
		return failure.InvalidBlockchain{
			HaveBlockchain: network.Blockchain,
			WantBlockchain: c.network.Blockchain,
			Description:    failure.NewDescription(blockchainUnknown),
		}
	}
	if network.Network != c.network.Network {
		return failure.InvalidNetwork{
			HaveNetwork: network.Network,
			WantNetwork: c.network.Network,
			Description: failure.NewDescription(networkUnknown),
		}
	}
	return nil
}
