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
	"github.com/optakt/flow-dps/rosetta/meta"
)

var (
	ErrorInternal           = meta.ErrorDefinition{Code: 1, Message: "internal error", Retriable: false}
	ErrorInvalidFormat      = meta.ErrorDefinition{Code: 2, Message: "invalid request format", Retriable: false}
	ErrorInvalidNetwork     = meta.ErrorDefinition{Code: 3, Message: "invalid network identifier", Retriable: false}
	ErrorInvalidBlock       = meta.ErrorDefinition{Code: 4, Message: "invalid block identifier", Retriable: false}
	ErrorInvalidTransaction = meta.ErrorDefinition{Code: 6, Message: "invalid transaction identifier", Retriable: false}
	ErrorInvalidAccount     = meta.ErrorDefinition{Code: 5, Message: "invalid account identifier", Retriable: false}
	ErrorInvalidCurrency    = meta.ErrorDefinition{Code: 6, Message: "invalid currency identifier", Retriable: false}
	ErrorUnknownBlock       = meta.ErrorDefinition{Code: 7, Message: "unknown block identifier", Retriable: true}
	ErrorUnknownCurrency    = meta.ErrorDefinition{Code: 8, Message: "unknown currency identifier", Retriable: false}
)
