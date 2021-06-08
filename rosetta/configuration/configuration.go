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
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type Configuration struct {
	network    identifier.Network
	version    object.Version
	statuses   []object.StatusDefinition
	operations []string
	errors     []object.ErrorDefinition
}

func New(chain flow.ChainID) *Configuration {

	network := identifier.Network{
		Blockchain: dps.FlowBlockchain,
		Network:    chain.String(),
	}

	// TODO: Find a way to inject the Flow Go dependency version from the
	// `go.mod` file and the middleware version from the repository tag:
	// => https://github.com/optakt/flow-dps/issues/151

	version := object.Version{
		RosettaVersion:    "1.4.10",
		NodeVersion:       "1.17.4",
		MiddlewareVersion: "0.0.0",
	}

	statuses := []object.StatusDefinition{
		{Status: "COMPLETED", Successful: true},
	}

	operations := []string{
		"TRANSFER",
	}

	errors := []object.ErrorDefinition{
		{Code: object.AnyCode, Message: object.AnyMessage, Retriable: object.AnyRetriable},
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

func (c *Configuration) Version() object.Version {
	return c.version
}

func (c *Configuration) Statuses() []object.StatusDefinition {
	return c.statuses
}

func (c *Configuration) Operations() []string {
	return c.operations
}

func (c *Configuration) Errors() []object.ErrorDefinition {
	return c.errors
}
