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

package response

import (
	"github.com/optakt/flow-dps/rosetta/meta"
)

// Options implements the successful response schema for the /network/options endpoint.
// See https://www.rosetta-api.org/docs/NetworkApi.html#200---ok-1
type Options struct {
	Version meta.Version `json:"version"`
	Allow   OptionsAllow `json:"allow"`
}

// OptionsAllow specifies supported Operation statuses, Operation types, and all possible
// error statuses. It is returned by the /network/options endpoint.
type OptionsAllow struct {
	OperationStatuses       []meta.StatusDefinition `json:"operation_statuses"`
	OperationTypes          []string                `json:"operation_types"`
	Errors                  []meta.ErrorDefinition  `json:"errors"`
	HistoricalBalanceLookup bool                    `json:"historical_balance_lookup"`
	CallMethods             []string                `json:"call_methods"`       // not used
	BalanceExemptions       []struct{}              `json:"balance_exemptions"` // not used
	MempoolCoins            bool                    `json:"mempool_coins"`
}
