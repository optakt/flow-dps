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

// TODO: consider separating Data and Construction API
// TODO: when the construction API crystalizes, fix the error returns
// TODO: as a WIP thing - treat the sender as the proposer and the payer. Consider allowing more granular control.

// Construction implements the Rosetta Construction API specification.
// See https://www.rosetta-api.org/docs/construction_api_introduction.html
type Construction struct {
	config    Configuration
	parser    Parser
	retriever Retriever
}

// NewConstruction creates a new instance of the Construction API using the given configuration
// to handle transaction construction requests.
// TODO: check - we only need the retriever for the single thing - get the reference block ID
func NewConstruction(config Configuration, parser Parser, retriever Retriever) *Construction {

	c := Construction{
		config:    config,
		parser:    parser,
		retriever: retriever,
	}

	return &c
}
