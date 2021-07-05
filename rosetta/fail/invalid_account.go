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

package fail

import (
	"github.com/optakt/flow-dps/rosetta/configuration"
)

// InvalidAccount error is returned when the specified account identifier is invalid.
// Account identifier is considered invalid when:
//	- account address is not a valid hex-encoded string
//	- account address fails the Flow chain address generator check
type InvalidAccount struct {
	Description string
	Details     []Detail
}

// Error returns the textual representation of the InvalidAccount error.
func (i InvalidAccount) Error() string {
	return "invalid account"
}

// RosettaError returns the error information in a Rosetta-compatible format.
func (i InvalidAccount) RosettaError() Error {
	return newError(configuration.ErrorInvalidAccount, i.Description, i.Details...)
}
