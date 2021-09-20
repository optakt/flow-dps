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
	"errors"

	"github.com/go-playground/validator/v10"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/request"
)

// ErrInvalidValidation is a sentinel error for any error not being caused by a given input failing validation.
var ErrInvalidValidation = errors.New("invalid validation input")

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

	blockchainFailTag = "blockchain"
	networkFailTag    = "network"
)

func newRequestValidator(config Configuration) *validator.Validate {

	validate := validator.New()

	// Register custom validators for known types.
	// We register a single type per validator, so we can safely perform type
	// assertion of the provided `validator.StructLevel` to the correct type.
	// Validation library stores the validators in a map, so if multiple
	// validators are registered for a single type, only the last one will be used.
	validate.RegisterStructValidation(blockValidator, identifier.Block{})
	validate.RegisterStructValidation(accountValidator, identifier.Account{})
	validate.RegisterStructValidation(transactionValidator, identifier.Transaction{})
	validate.RegisterStructValidation(networkValidator(config), identifier.Network{})

	// Register custom top-level validators. These validate the entire request
	// object, compared to the ones above which validate a specific type
	// within the request. This way we can validate some standard types (strings)
	// or complex ones (array of currencies) in a structured way.
	validate.RegisterStructValidation(balanceValidator, request.Balance{})
	validate.RegisterStructValidation(parseValidator, request.Parse{})
	validate.RegisterStructValidation(combineValidator, request.Combine{})
	validate.RegisterStructValidation(submitValidator, request.Submit{})
	validate.RegisterStructValidation(hashValidator, request.Hash{})

	return validate
}

// Request runs the registered validators on the provided request.
// It either returns a typed error with contextual information, or a plain error
// describing what failed.
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
		return ErrInvalidValidation
	}

	// Process validation errors we have found. Return the first one we encounter.
	// Validator library doesn't offer a sophisticated mechanism for passing detailed
	// error information from the lower validation layers, so we use the `tag` field
	// to identify what went wrong.
	// In the case of blockchain/network misconfigurtion, we also use the `param` field
	// to pass the acceptable value for the field.

	verr := err.(validator.ValidationErrors)[0]

	switch verr.Tag() {
	case blockLength:
		// Block hash has incorrect length.
		blockHash, _ := verr.Value().(string)
		return failure.InvalidBlockHash{
			Description: failure.NewDescription(blockLength),
			WantLength:  rosetta.HexIDSize,
			HaveLength:  len(blockHash),
		}

	case addressLength:
		// Account address has incorrect length.
		address, _ := verr.Value().(string)
		return failure.InvalidAccountAddress{
			Description: failure.NewDescription(addressLength),
			WantLength:  rosetta.HexAddressSize,
			HaveLength:  len(address),
		}

	case txLength:
		// Transaction hash has incorrect length.
		txHash, _ := verr.Value().(string)
		return failure.InvalidTransactionHash{
			Description: failure.NewDescription(txLength),
			WantLength:  rosetta.HexIDSize,
			HaveLength:  len(txHash),
		}

	case networkFailTag:
		// Network field is not referring to the network for which we are configured.
		network, _ := verr.Value().(identifier.Network)
		return failure.InvalidNetwork{
			HaveNetwork: network.Network,
			WantNetwork: verr.Param(),
			Description: failure.NewDescription(networkUnknown),
		}

	case blockchainFailTag:
		// Blockchain field is not referring to the blockchain for which we are configured.
		network, _ := verr.Value().(identifier.Network)
		return failure.InvalidBlockchain{
			HaveBlockchain: network.Blockchain,
			WantBlockchain: verr.Param(),
			Description:    failure.NewDescription(blockchainUnknown),
		}

	default:
		return errors.New(verr.Tag())
	}
}

// networkValidator verifies that the request is valid for the configured network.
// Network and blockchain fields must be populated and valid for the current
// running instance of the DPS. For example, a request cannot address a Testnet
// network while we are running on top of a Mainnet index.
func networkValidator(config Configuration) func(validator.StructLevel) {
	return func(sl validator.StructLevel) {
		network := sl.Current().Interface().(identifier.Network)

		if network.Blockchain == "" {
			sl.ReportError(network.Blockchain, blockchainField, blockchainField, blockchainEmpty, "")
		}
		if network.Network == "" {
			sl.ReportError(network.Network, networkField, networkField, networkEmpty, "")
		}

		err := config.Check(network)
		// Check returns a typed error, which we can use to identify which field was invalid.
		// We will use the `tag` field to communicate this, and the `param` field to pass the acceptable
		// value to the main validation function.
		var ibErr failure.InvalidBlockchain
		if errors.As(err, &ibErr) {
			sl.ReportError(network, blockchainField, blockchainField, blockchainFailTag, ibErr.WantBlockchain)
		}
		var inErr failure.InvalidNetwork
		if errors.As(err, &inErr) {
			sl.ReportError(network, networkField, networkField, networkFailTag, inErr.WantNetwork)
		}
	}
}

// blockValidator ensures that, if the block hash is provided, it has the correct length.
func blockValidator(sl validator.StructLevel) {
	rosBlockID := sl.Current().Interface().(identifier.Block)
	if rosBlockID.Hash != "" && len(rosBlockID.Hash) != rosetta.HexIDSize {
		sl.ReportError(rosBlockID.Hash, blockHashField, blockHashField, blockLength, "")
	}
}

// accountValidator ensures that the account address field is populated and has correct length.
func accountValidator(sl validator.StructLevel) {
	rosAccountID := sl.Current().Interface().(identifier.Account)
	if rosAccountID.Address == "" {
		sl.ReportError(rosAccountID.Address, addressField, addressField, addressEmpty, "")
	}
	if len(rosAccountID.Address) != rosetta.HexAddressSize {
		sl.ReportError(rosAccountID.Address, addressField, addressField, addressLength, "")
	}
}

// transactionValidator ensures that the transaction identifier is populated and has correct length.
func transactionValidator(sl validator.StructLevel) {
	rosTxID := sl.Current().Interface().(identifier.Transaction)
	if rosTxID.Hash == "" {
		sl.ReportError(rosTxID.Hash, txField, txField, txHashEmpty, "")
	}
	if len(rosTxID.Hash) != rosetta.HexIDSize {
		sl.ReportError(rosTxID.Hash, txField, txField, txLength, "")
	}
}

// balanceValidator ensures that the provided Balance request has a non-empty currency list, and that
// all provided currencies have the `symbol` field populated.
func balanceValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(request.Balance)
	if len(req.Currencies) == 0 {
		sl.ReportError(req.Currencies, currencyField, currencyField, currenciesEmpty, "")
	}
	for _, currency := range req.Currencies {
		if currency.Symbol == "" {
			sl.ReportError(currency.Symbol, symbolField, symbolField, symbolEmpty, "")
		}
	}
}

// parseValidator ensures that the provided Parse request has a non-empty transaction field.
func parseValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(request.Parse)
	if req.Transaction == "" {
		sl.ReportError(req.Transaction, transactionField, transactionField, txBodyEmpty, "")
	}
}

// combineValidator ensures that the provided Combine request has a non-empty transaction field, and
// that the signature list is not empty.
func combineValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(request.Combine)
	if req.UnsignedTransaction == "" {
		sl.ReportError(req.UnsignedTransaction, transactionField, transactionField, txBodyEmpty, "")
	}

	if len(req.Signatures) == 0 {
		sl.ReportError(req.Signatures, signaturesField, signaturesField, signaturesEmpty, "")
	}
}

// submitValidator ensures that the provided Submit request has a non-empty transaction field.
func submitValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(request.Submit)
	if req.SignedTransaction == "" {
		sl.ReportError(req.SignedTransaction, transactionField, transactionField, txBodyEmpty, "")
	}
}

// hashValidator ensures that the provided Hash request has a non-empty transaction field.
func hashValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(request.Hash)
	if req.SignedTransaction == "" {
		sl.ReportError(req.SignedTransaction, transactionField, transactionField, txBodyEmpty, "")
	}
}
