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

package mapper

// Status is a representation of the state machine's status.
type Status uint8

// The following is an enumeration of all possible statuses the
// state machine can have.
const (
	StatusEmpty Status = iota + 1
	StatusUpdating
	StatusMatched
	StatusCollected
	StatusIndexed
	StatusForwarded
)

// String implements the Stringer interface.
func (s Status) String() string {
	switch s {
	case StatusEmpty:
		return "empty"
	case StatusUpdating:
		return "updating"
	case StatusMatched:
		return "matched"
	case StatusCollected:
		return "collected"
	case StatusIndexed:
		return "indexed"
	case StatusForwarded:
		return "forwarded"
	default:
		return "invalid"
	}
}
