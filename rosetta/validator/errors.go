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
	unknownBlockchain    = "network identifier has unknown blockchain field"
	blockchainEmpty      = "blockchain identifier has empty blockchain field"
	unknownNetwork       = "network identifier has unknown network field"
	networkEmpty         = "blockchain identifier has empty network field"
	blockInvalid         = "block hash is not a valid hex-encoded string"
	blockNotFull         = "block identifier needs both fields filled for this request"
	blockLength          = "block identifier has invalid hash field length"
	blockTooLow          = "block index is below first indexed height"
	blockTooHigh         = "block index is above last indexed height"
	blockMismatch        = "block hash mismatches with authoritative hash for index"
	addressEmpty         = "account identifier has empty address field"
	addressInvalid       = "account address is not a valid hex-encoded string"
	addressMisconfigured = "account address is not valid for configured chain"
	addressLength        = "account identifier has invalid address field length"
	currenciesEmpty      = "currency identifier list is empty"
	symbolEmpty          = "currency identifier has empty symbol field"
	symbolUnknown        = "currency symbol is unknown"
	decimalsMismatch     = "currency decimals mismatch with authoritative decimals for symbol"
	txHashEmpty          = "transaction identifier has empty hash field"
	txHashInvalid        = "transaction hash is not a valid hex-encoded string"
	txLength             = "transaction identifier has invalid hash field length"
	txBodyEmpty          = "transaction text is empty"
	signaturesEmpty      = "signature list is empty"
)
