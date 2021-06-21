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

package identifier

// Account uniquely identifies an account within a network. All fields in the
// account identifier are utilized to determine uniqueness, including the
// metadata field, if populated. We don't use sub-accounts in this
// implementation for now, though we will probably have to add it to support
// staking on Coinbase in the future.
type Account struct {
	Address string `json:"address"`
}
