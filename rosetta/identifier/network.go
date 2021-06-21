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

// Network specifies which network a particular object is associated with. The
// blockchain field is always set to `flow` and the network is always set to
// `mainnet`.
//
// We are ommitting the `SubNetwork` field for now, but we could use it in the
// future to distinguish between the networks of different sporks (i.e.
// `candidate4` or `mainnet-5`).
type Network struct {
	Blockchain string `json:"blockchain"`
	Network    string `json:"network"`
}
