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

package convert

import (
	"github.com/onflow/flow-go/model/flow"
)

// IDToHash converts a flow Identifier into a byte slice.
func IDToHash(id flow.Identifier) []byte {
	hash := make([]byte, 32)
	copy(hash, id[:])

	return hash
}

// CommitToHash converts a flow StateCommitment into a byte slice.
func CommitToHash(commit flow.StateCommitment) []byte {
	hash := make([]byte, 32)
	copy(hash, commit[:])

	return hash
}
