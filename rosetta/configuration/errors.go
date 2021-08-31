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
	"github.com/optakt/flow-dps/rosetta/meta"
)

var (
	ErrorInternal           = meta.ErrorDefinition{Code: 1, Message: "internal error", Retriable: false}
	ErrorInvalidEncoding    = meta.ErrorDefinition{Code: 2, Message: "invalid request encoding", Retriable: false}
	ErrorInvalidFormat      = meta.ErrorDefinition{Code: 3, Message: "invalid request format", Retriable: false}
	ErrorInvalidNetwork     = meta.ErrorDefinition{Code: 4, Message: "invalid network identifier", Retriable: false}
	ErrorInvalidAccount     = meta.ErrorDefinition{Code: 5, Message: "invalid account identifier", Retriable: false}
	ErrorInvalidCurrency    = meta.ErrorDefinition{Code: 6, Message: "invalid currency identifier", Retriable: false}
	ErrorInvalidBlock       = meta.ErrorDefinition{Code: 7, Message: "invalid block identifier", Retriable: false}
	ErrorInvalidTransaction = meta.ErrorDefinition{Code: 8, Message: "invalid transaction identifier", Retriable: false}
	ErrorUnknownBlock       = meta.ErrorDefinition{Code: 9, Message: "unknown block identifier", Retriable: true}
	ErrorUnknownCurrency    = meta.ErrorDefinition{Code: 10, Message: "unknown currency identifier", Retriable: false}
	ErrorUnknownTransaction = meta.ErrorDefinition{Code: 11, Message: "unknown block transaction", Retriable: false}

	// Construction API specific errors.
	ErrorInvalidTransactionIntent = meta.ErrorDefinition{Code: 12, Message: "invalid transaction intent", Retriable: false}
	ErrorInvalidAuthorizers       = meta.ErrorDefinition{Code: 13, Message: "invalid transaction authorizers", Retriable: false}
	ErrorInvalidPayer             = meta.ErrorDefinition{Code: 14, Message: "invalid transaction payer", Retriable: false}
	ErrorInvalidProposer          = meta.ErrorDefinition{Code: 15, Message: "invalid transaction proposer", Retriable: false}
	ErrorInvalidScript            = meta.ErrorDefinition{Code: 16, Message: "invalid transaction script", Retriable: false}
	ErrorInvalidArguments         = meta.ErrorDefinition{Code: 17, Message: "invalid transaction arguments", Retriable: false}
	ErrorInvalidAmount            = meta.ErrorDefinition{Code: 18, Message: "invalid transaction amount", Retriable: false}
	ErrorInvalidReceiver          = meta.ErrorDefinition{Code: 19, Message: "invalid transaction recipient", Retriable: false}
	ErrorInvalidSignature         = meta.ErrorDefinition{Code: 20, Message: "invalid transaction signature", Retriable: false}
	ErrorInvalidProposalKey       = meta.ErrorDefinition{Code: 21, Message: "invalid transaction proposal key", Retriable: false}
	ErrorInvalidAuthorizerKey     = meta.ErrorDefinition{Code: 22, Message: "invalid transaction authorizer key", Retriable: false}
)
