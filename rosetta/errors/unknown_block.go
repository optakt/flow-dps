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

package errors

// UnknownBlock error is returned when the requested block is not known, because
// the height of the requested block is larger than the height of the last known block.
type UnknownBlock struct {
	Description string
	Details     []Detail
}

// Error returns the textual representation of the UnknownBlock error.
func (u UnknownBlock) Error() string {
	return "unknown block"
}

// RosettaError returns the error information in a Rosetta-compatible format.
func (u UnknownBlock) RosettaError() Error {
	return newError(ErrorUnknownBlock, u.Description, u.Details...)
}
