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

package rosetta

// TODO: Sender is also the proposer and the payer. Consider allowing more granular control.
// => https://github.com/optakt/flow-dps/issues/331

// TODO: Specify Gas limit for the transaction.
// => https://github.com/optakt/flow-dps/issues/332

// Construction implements the Rosetta Construction API specification.
// See https://www.rosetta-api.org/docs/construction_api_introduction.html
type Construction struct {
	config   Configuration
	transact Transactor

	// Retrieve is used to get the latest block ID. This is needed since
	// transactions require a reference block ID, so that their validity
	// or expiration can be determined.
	retrieve Retriever
}

// NewConstruction creates a new instance of the Construction API using the given configuration
// to handle transaction construction requests.
func NewConstruction(config Configuration, transact Transactor) *Construction {

	c := Construction{
		config:   config,
		transact: transact,
	}

	return &c
}
