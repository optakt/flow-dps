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

// Error descriptions for common errors.
const (
	// Transaction actor errors.
	authorizersInvalid = "invalid number of authorizers"
	payerInvalid       = "invalid transaction payer"
	proposerInvalid    = "invalid transaction proposer"

	// Transaction signature errors.
	signerInvalid           = "invalid signer account"
	sigInvalid              = "provided signature is not valid"
	sigEncoding             = "invalid signature payload"
	payloadSigFound         = "unexpected payload signature found"
	envelopeSigFound        = "unexpected envelope signature found"
	envelopeSigCountInvalid = "unexpected number of envelope signatures"
	sigCountInvalid         = "invalid number of signatures"
	sigAlgoInvalid          = "invalid signature algorithm"

	// Transaction script errors.
	scriptInvalid       = "transaction text is not valid token transfer script"
	scriptArgsInvalid   = "invalid number of arguments"
	amountUnparseable   = "could not parse transaction amount"
	amountInvalid       = "invalid amount"
	receiverUnparseable = "could not parse transaction receiver address"

	// Operations/intent errors.
	opsInvalid          = "invalid number of operations"
	opsAmountsMismatch  = "transfer amounts do not match"
	currenciesInvalid   = "invalid currencies found"
	opAmountUnparseable = "could not parse amount"
	opTypeInvalid       = "only transfer operations are supported"
	keyInvalid          = "invalid account key"
)
