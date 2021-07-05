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

// UnknownCurrency error is returned when the requested currency is not known.
// This happens when the specified currency symbol is not configured.
type UnknownCurrency struct {
	Description string
	Details     []Detail
}

// Error returns the textual representation of the UnknownCurrency error.
func (u UnknownCurrency) Error() string {
	return "unknown currency"
}

// RosettaError returns the error information in a Rosetta-compatible format.
func (u UnknownCurrency) RosettaError() Error {
	return newError(ErrorUnknownCurrency, u.Description, u.Details...)
}
