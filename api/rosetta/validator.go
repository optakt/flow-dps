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
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/optakt/flow-dps/rosetta/identifier"
)

var (
	errInvalidValidation = errors.New("invalid validation input")

	errBlockLength   = errors.New(blockLength)
	errAddressLength = errors.New(addressLength)
	errTxLength      = errors.New(txLength)
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

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {

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
	v.RegisterStructValidation(balanceValidator, BalanceRequest{})
	v.RegisterStructValidation(parseValidator, ParseRequest{})
	v.RegisterStructValidation(combineValidator, CombineRequest{})
	v.RegisterStructValidation(submitValidator, SubmitRequest{})
	v.RegisterStructValidation(hashValidator, HashRequest{})

	validator := Validator{
		validate: v,
	}
	return &validator
}

func (v *Validator) Validate(request interface{}) error {

	err := v.validate.Struct(request)
	// If validation passed ok - we're done.
	if err == nil {
		return nil
	}

	// InvalidValidationError is returned by the validation library in cases of invalid usage,
	// more precisely, passing a non-struct to `validate.Struct()` method.
	_, ok := err.(*validator.InvalidValidationError)
	if ok {
		return errInvalidValidation
	}

	// Process validation errors we have found. Return the first one we encounter.
	errs := err.(validator.ValidationErrors)
	msg := errs[0].Tag()

	switch msg {
	case blockLength:
		return errBlockLength
	case addressLength:
		return errAddressLength
	case txLength:
		return errTxLength
	default:
		return fmt.Errorf(msg)
	}
}

func blockValidator(sl validator.StructLevel) {
	rosBlockID := sl.Current().Interface().(identifier.Block)
	if rosBlockID.Hash != "" && len(rosBlockID.Hash) != hexIDSize {
		sl.ReportError(rosBlockID.Hash, blockHashField, blockHashField, blockLength, "")
	}
}

func networkValidator(sl validator.StructLevel) {
	network := sl.Current().Interface().(identifier.Network)
	if network.Blockchain == "" {
		sl.ReportError(network.Blockchain, blockchainField, blockchainField, blockchainEmpty, "")
	}
	if network.Network == "" {
		sl.ReportError(network.Network, networkField, networkField, networkEmpty, "")
	}
}

func accountValidator(sl validator.StructLevel) {
	rosAccountID := sl.Current().Interface().(identifier.Account)
	if rosAccountID.Address == "" {
		sl.ReportError(rosAccountID.Address, addressField, addressField, addressEmpty, "")
	}
	if len(rosAccountID.Address) != hexAddressSize {
		sl.ReportError(rosAccountID.Address, addressField, addressField, addressLength, "")
	}
}

func transactionValidator(sl validator.StructLevel) {
	rosTxID := sl.Current().Interface().(identifier.Transaction)
	if rosTxID.Hash == "" {
		sl.ReportError(rosTxID.Hash, txField, txField, txHashEmpty, "")
	}
	if len(rosTxID.Hash) != hexIDSize {
		sl.ReportError(rosTxID.Hash, txField, txField, txLength, "")
	}
}

func balanceValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(BalanceRequest)
	if len(req.Currencies) == 0 {
		sl.ReportError(req.Currencies, currencyField, currencyField, currenciesEmpty, "")
	}
	for _, currency := range req.Currencies {
		if currency.Symbol == "" {
			sl.ReportError(currency.Symbol, symbolField, symbolField, symbolEmpty, "")
		}
	}
}

func parseValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(ParseRequest)
	if req.Transaction == "" {
		sl.ReportError(req.Transaction, transactionField, transactionField, txBodyEmpty, "")
	}
}

func combineValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(CombineRequest)
	if req.UnsignedTransaction == "" {
		sl.ReportError(req.UnsignedTransaction, transactionField, transactionField, txBodyEmpty, "")
	}

	if len(req.Signatures) == 0 {
		sl.ReportError(req.Signatures, signaturesField, signaturesField, signaturesEmpty, "")
	}
}

func submitValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(SubmitRequest)
	if req.SignedTransaction == "" {
		sl.ReportError(req.SignedTransaction, transactionField, transactionField, txBodyEmpty, "")
	}
}

func hashValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(HashRequest)
	if req.SignedTransaction == "" {
		sl.ReportError(req.SignedTransaction, transactionField, transactionField, txBodyEmpty, "")
	}
}
