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

// Currency is composed of a canonical symbol and decimals. This decimals value
// is used to convert an amount value from atomic units (such as satoshis) to
// standard units (such as bitcoins). As monetary values in Flow are provided as
// an unsigned fixed point value with 8 decimals, we simply use the full integer
// with 8 decimals in the currency struct. The symbol is always `FLOW`.
//
// An example of metadata given in the Rosetta API documentation is `Issuer`.
type Currency struct {
	Symbol   string `json:"symbol"`
	Decimals uint   `json:"decimals,omitempty"`
}
