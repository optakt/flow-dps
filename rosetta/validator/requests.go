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

import (
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

// Field names are displayed if the code deals with the plain `error` types instead of the structured validation errors.
// When dealing with plain errors, the validation error reads "Field validation for 'FieldName' failed on the 'X' tag".
// Since in our code we are dealing with structured errors, these field names are not actually used.
// However, they are mandatory arguments for the `ReportError` method from the validator library.
const (
	blockHashField   = "block_hash"
	blockchainField  = "blockchain"
	networkField     = "network"
	addressField     = "address"
	txField          = "transaction_id"
	currencyField    = "currency"
	symbolField      = "symbol"
	transactionField = "transaction"
	signaturesField  = "signatures"
)

func newRequestValidator() *validator.Validate {

	v := validator.New()

	// Register custom validators for known types.
	// We register a single type per validator, so we can safely perform type
	// assertion of the provided `validator.StructLevel` to the correct type.
	v.RegisterStructValidation(blockValidator, identifier.Block{})
	v.RegisterStructValidation(networkValidator, identifier.Network{})
	v.RegisterStructValidation(accountValidator, identifier.Account{})
	v.RegisterStructValidation(transactionValidator, identifier.Transaction{})

	// Register custom top-level validators. These validate the entire request
	// object, compared to the ones above which validate a specific type
	// within the request. This way we can validate some standard types (strings)
	// or complex ones (array of currencies) in a structured way.
	v.RegisterStructValidation(balanceValidator, rosetta.BalanceRequest{})
	v.RegisterStructValidation(parseValidator, rosetta.ParseRequest{})
	v.RegisterStructValidation(combineValidator, rosetta.CombineRequest{})
	v.RegisterStructValidation(submitValidator, rosetta.SubmitRequest{})
	v.RegisterStructValidation(hashValidator, rosetta.HashRequest{})

	return v
}

func (v *Validator) Request(request interface{}) error {

	err := v.validate.Struct(request)
	// If validation passed ok - we're done.
	if err == nil {
		return nil
	}

	// InvalidValidationError is returned by the validation library in cases of invalid usage,
	// more precisely, passing a non-struct to `validate.Struct()` method.
	_, ok := err.(*validator.InvalidValidationError)
	if ok {
		return rosetta.ErrInvalidValidation
	}

	// Process validation errors we have found. Return the first one we encounter.
	errs := err.(validator.ValidationErrors)
	msg := errs[0].Tag()

	switch msg {
	case rosetta.BlockLength:
		return rosetta.ErrBlockLength
	case rosetta.AddressLength:
		return rosetta.ErrAddressLength
	case rosetta.TxLength:
		return rosetta.ErrTxLength
	default:
		return fmt.Errorf(msg)
	}
}

func blockValidator(sl validator.StructLevel) {
	rosBlockID := sl.Current().Interface().(identifier.Block)
	if rosBlockID.Hash != "" && len(rosBlockID.Hash) != rosetta.HexIDSize {
		sl.ReportError(rosBlockID.Hash, blockHashField, blockHashField, rosetta.BlockLength, "")
	}
}

func networkValidator(sl validator.StructLevel) {
	network := sl.Current().Interface().(identifier.Network)
	if network.Blockchain == "" {
		sl.ReportError(network.Blockchain, blockchainField, blockchainField, rosetta.BlockchainEmpty, "")
	}
	if network.Network == "" {
		sl.ReportError(network.Network, networkField, networkField, rosetta.NetworkEmpty, "")
	}
}

func accountValidator(sl validator.StructLevel) {
	rosAccountID := sl.Current().Interface().(identifier.Account)
	if rosAccountID.Address == "" {
		sl.ReportError(rosAccountID.Address, addressField, addressField, rosetta.AddressEmpty, "")
	}
	if len(rosAccountID.Address) != rosetta.HexAddressSize {
		sl.ReportError(rosAccountID.Address, addressField, addressField, rosetta.AddressLength, "")
	}
}

func transactionValidator(sl validator.StructLevel) {
	rosTxID := sl.Current().Interface().(identifier.Transaction)
	if rosTxID.Hash == "" {
		sl.ReportError(rosTxID.Hash, txField, txField, rosetta.TxHashEmpty, "")
	}
	if len(rosTxID.Hash) != rosetta.HexIDSize {
		sl.ReportError(rosTxID.Hash, txField, txField, rosetta.TxLength, "")
	}
}

func balanceValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(rosetta.BalanceRequest)
	if len(req.Currencies) == 0 {
		sl.ReportError(req.Currencies, currencyField, currencyField, rosetta.CurrenciesEmpty, "")
	}
	for _, currency := range req.Currencies {
		if currency.Symbol == "" {
			sl.ReportError(currency.Symbol, symbolField, symbolField, rosetta.SymbolEmpty, "")
		}
	}
}

func parseValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(rosetta.ParseRequest)
	if req.Transaction == "" {
		sl.ReportError(req.Transaction, transactionField, transactionField, rosetta.TxBodyEmpty, "")
	}
}

func combineValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(rosetta.CombineRequest)
	if req.UnsignedTransaction == "" {
		sl.ReportError(req.UnsignedTransaction, transactionField, transactionField, rosetta.TxBodyEmpty, "")
	}

	if len(req.Signatures) == 0 {
		sl.ReportError(req.Signatures, signaturesField, signaturesField, rosetta.SignaturesEmpty, "")
	}
}

func submitValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(rosetta.SubmitRequest)
	if req.SignedTransaction == "" {
		sl.ReportError(req.SignedTransaction, transactionField, transactionField, rosetta.TxBodyEmpty, "")
	}
}

func hashValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(rosetta.HashRequest)
	if req.SignedTransaction == "" {
		sl.ReportError(req.SignedTransaction, transactionField, transactionField, rosetta.TxBodyEmpty, "")
	}
}
