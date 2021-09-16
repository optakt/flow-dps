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

package object

// Signature contains the information about a transaction signature.
type Signature struct {
	SigningPayload SigningPayload `json:"signing_payload"`
	SignatureType  string         `json:"signature_type"`
	HexBytes       string         `json:"hex_bytes"`
	PublicKey      PublicKey      `json:"public_key"`
}

// PublicKey represents a public key used in a transaction signature.
type PublicKey struct {
	HexBytes  string `json:"hex_bytes"`
	CurveType string `json:"curve_type"`
}
