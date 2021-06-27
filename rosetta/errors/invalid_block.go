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

// InvalidBlock error is returned when the specified block identifier is invalid.
// Block identifier is considered invalid when:
//	- block hash is not a valid hex-encoded string
//	- block index is below the first indexed block
//	- block hash does not match the known block hash
type InvalidBlock struct {
	Description string
	Details     []Detail
}

// Error returns the textual representation of the InvalidBlock error.
func (i InvalidBlock) Error() string {
	return "invalid block"
}

// RosettaError returns the error information in a Rosetta-compatible format.
func (i InvalidBlock) RosettaError() Error {
	return newError(ErrorInvalidBlock, i.Description, i.Details...)
}
