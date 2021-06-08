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

package configuration

import (
	"fmt"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/meta"
)

type Configuration struct {
	network    identifier.Network
	version    meta.Version
	statuses   []meta.StatusDefinition
	operations []string
	errors     []meta.ErrorDefinition
}

func New(chain flow.ChainID) *Configuration {

	network := identifier.Network{
		Blockchain: dps.FlowBlockchain,
		Network:    chain.String(),
	}

	// TODO: Find a way to inject the Flow Go dependency version from the
	// `go.mod` file and the middleware version from the repository tag:
	// => https://github.com/optakt/flow-dps/issues/151

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
		ErrorInvalidNetwork,
		ErrorInvalidBlock,
		ErrorInvalidAccount,
		ErrorInvalidTransaction,
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

func (c *Configuration) Network() identifier.Network {
	return c.network
}

func (c *Configuration) Version() meta.Version {
	return c.version
}

func (c *Configuration) Statuses() []meta.StatusDefinition {
	return c.statuses
}

func (c *Configuration) Operations() []string {
	return c.operations
}

func (c *Configuration) Errors() []meta.ErrorDefinition {
	return c.errors
}

func (c *Configuration) Check(network identifier.Network) error {
	if network.Blockchain != c.network.Blockchain {
		return failure.InvalidNetwork{
			Blockchain: network.Blockchain,
			Network:    network.Network,
			Message:    fmt.Sprintf("invalid network identifier blockchain (have: %s, want: %s)", network.Blockchain, c.network.Blockchain),
		}
	}
	if network.Network != c.network.Network {
		return failure.InvalidNetwork{
			Blockchain: network.Blockchain,
			Network:    network.Network,
			Message:    fmt.Sprintf("invalid network identifier network (have: %s, want: %s)", network.Network, c.network.Network),
		}
	}
	return nil
}
