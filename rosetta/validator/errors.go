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

package validator

// Error descriptions for common errors.
const (
	unknownBlockchain = "network identifier has unknown blockchain field"
	blockchainEmpty   = "blockchain identifier has empty blockchain field"
	unknownNetwork    = "network identifier has unknown network field"
	networkEmpty      = "blockchain identifier has empty network field"
	blockNotFull      = "block identifier needs both fields filled for this request"
	blockLength       = "block identifier has invalid hash field length"
	addressEmpty      = "account identifier has empty address field"
	addressLength     = "account identifier has invalid address field length"
	currenciesEmpty   = "currency identifier list is empty"
	symbolEmpty       = "currency identifier has empty symbol field"
	txHashEmpty       = "transaction identifier has empty hash field"
	txLength          = "transaction identifier has invalid hash field length"
	txBodyEmpty       = "transaction text is empty"
	signaturesEmpty   = "signature list is empty"
)
