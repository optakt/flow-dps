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

package transactions

// Parser has the capabilities to determine transaction intent from an array
// of Rosetta operations, create a Flow transaction from a transaction intent
// and transate a Flow transaction back to an array of Rosetta operations.
type Parser struct {
	validate Validator
	generate Generator
}

// NewParser creates a new transaction Parser to handle constructing
// and parsing transactions.
func NewParser(validate Validator, generate Generator) *Parser {

	p := Parser{
		validate: validate,
		generate: generate,
	}

	return &p
}
