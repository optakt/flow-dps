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

package rosetta

import (
	"github.com/go-playground/validator/v10"

	"github.com/optakt/flow-dps/rosetta/identifier"
)

// Field names needed by the validator library, not needed or used by our code at the moment.
const (
	blockHashField  = "block_hash"
	blockchainField = "blockchain"
	networkField    = "network"
	addressField    = "address"
	txField         = "transaction_id"
	symbolField     = "symbol"
)

func blockValidator(sl validator.StructLevel) {
	rosBlockID, ok := sl.Current().Interface().(identifier.Block)
	if !ok {
		return
	}

	if rosBlockID.Hash != "" && len(rosBlockID.Hash) != hexIDSize {
		sl.ReportError(rosBlockID.Hash, blockHashField, blockHashField, blockLength, "")
	}
}

func networkValidator(sl validator.StructLevel) {
	network, ok := sl.Current().Interface().(identifier.Network)
	if !ok {
		return
	}

	if network.Blockchain == "" {
		sl.ReportError(network.Blockchain, blockchainField, blockchainField, blockchainEmpty, "")
	}
	if network.Network == "" {
		sl.ReportError(network.Network, networkField, networkField, networkEmpty, "")
	}
}

func accountValidator(sl validator.StructLevel) {
	rosAccountID, ok := sl.Current().Interface().(identifier.Account)
	if !ok {
		return
	}

	if rosAccountID.Address == "" {
		sl.ReportError(rosAccountID.Address, addressField, addressField, addressEmpty, "")
	}
	if len(rosAccountID.Address) != hexAddressSize {
		sl.ReportError(rosAccountID.Address, addressField, addressField, addressLength, "")
	}
}

func transactionValidator(sl validator.StructLevel) {
	rosTxID, ok := sl.Current().Interface().(identifier.Transaction)
	if !ok {
		return
	}

	if rosTxID.Hash == "" {
		sl.ReportError(rosTxID.Hash, txField, txField, txHashEmpty, "")
	}
	if len(rosTxID.Hash) != hexIDSize {
		sl.ReportError(rosTxID.Hash, txField, txField, txLength, "")
	}
}

func currencyValidator(sl validator.StructLevel) {
	currency, ok := sl.Current().Interface().(identifier.Currency)
	if !ok {
		return
	}

	if currency.Symbol == "" {
		sl.ReportError(currency.Symbol, symbolField, symbolField, symbolEmpty, "")
	}
}
