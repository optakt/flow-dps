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

package transactor

import (
	"testing"

	sdk "github.com/onflow/flow-go-sdk"
	"github.com/optakt/flow-dps/testing/mocks"
)

func BaselineTransactionParser(t *testing.T, opts ...func(parser *TransactionParser)) *TransactionParser {
	p := TransactionParser{
		tx:       sdk.NewTransaction(),
		validate: mocks.BaselineValidator(t),
		generate: mocks.BaselineGenerator(t),
		invoke:   mocks.BaselineInvoker(t),
	}

	for _, opt := range opts {
		opt(&p)
	}

	return &p
}

// FIXME: The naming of those funcs conflict with the transactor's, since they have the same dependencies :(
//        How can we name this in a consistent way that makes sense?
func ParseTransaction(tx *sdk.Transaction) func(*TransactionParser) {
	return func(parser *TransactionParser) {
		parser.tx = tx
	}
}

func ParseValidator(validate Validator) func(*TransactionParser) {
	return func(parser *TransactionParser) {
		parser.validate = validate
	}
}

func ParseGenerator(generate Generator) func(*TransactionParser) {
	return func(parser *TransactionParser) {
		parser.generate = generate
	}
}

func ParseInvoker(invoke Invoker) func(*TransactionParser) {
	return func(parser *TransactionParser) {
		parser.invoke = invoke
	}
}
