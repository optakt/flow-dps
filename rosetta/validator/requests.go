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
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

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

// Error descriptions for common errors.
const (
	unknownBlockchain = "network identifier has unknown blockchain field"
	unknownNetwork    = "network identifier has unknown network field"
)

func (v *Validator) newRequestValidator() *validator.Validate {

	vl := validator.New()

	// Register custom validators for known types.
	// We register a single type per validator, so we can safely perform type
	// assertion of the provided `validator.StructLevel` to the correct type.
	// Validation library stores the validators in a map, so if multiple
	// validators are registered for a single type, only the last one will be used.
	vl.RegisterStructValidation(blockValidator, identifier.Block{})
	vl.RegisterStructValidation(accountValidator, identifier.Account{})
	vl.RegisterStructValidation(transactionValidator, identifier.Transaction{})
	vl.RegisterStructValidation(v.networkValidator, identifier.Network{})

	// Register custom top-level validators. These validate the entire request
	// object, compared to the ones above which validate a specific type
	// within the request. This way we can validate some standard types (strings)
	// or complex ones (array of currencies) in a structured way.
	vl.RegisterStructValidation(balanceValidator, rosetta.BalanceRequest{})
	vl.RegisterStructValidation(parseValidator, rosetta.ParseRequest{})
	vl.RegisterStructValidation(combineValidator, rosetta.CombineRequest{})
	vl.RegisterStructValidation(submitValidator, rosetta.SubmitRequest{})
	vl.RegisterStructValidation(hashValidator, rosetta.HashRequest{})

	return vl
}

// Request will run the registered validators on the provided request.
// It will either return a typed error with contextual information, or a plain error
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
	case rosetta.BlockLength:
		// Block hash has incorrect length.
		blockHash, _ := verr.Value().(string)
		return failure.InvalidBlockHash{
			Description: failure.NewDescription(rosetta.BlockLength),
			WantLength:  rosetta.HexIDSize,
			HaveLength:  len(blockHash),
		}

	case rosetta.AddressLength:
		// Account address has incorrect length.
		address, _ := verr.Value().(string)
		return failure.InvalidAccountAddress{
			Description: failure.NewDescription(rosetta.AddressLength),
			WantLength:  rosetta.HexAddressSize,
			HaveLength:  len(address),
		}

	case rosetta.TxLength:
		// Transaction hash has incorrect length.
		txHash, _ := verr.Value().(string)
		return failure.InvalidTransactionHash{
			Description: failure.NewDescription(rosetta.TxLength),
			WantLength:  rosetta.HexIDSize,
			HaveLength:  len(txHash),
		}

	case networkFailTag:
		// Network field is not referring to the network for which we are configured.
		network, _ := verr.Value().(identifier.Network)
		return failure.InvalidNetwork{
			HaveNetwork: network.Network,
			WantNetwork: verr.Param(),
			Description: failure.NewDescription(unknownNetwork),
		}

	case blockchainFailTag:
		// Blockchain field is not referring to the blockchain for which we are configured.
		network, _ := verr.Value().(identifier.Network)
		return failure.InvalidBlockchain{
			HaveBlockchain: network.Blockchain,
			WantBlockchain: verr.Param(),
			Description:    failure.NewDescription(unknownBlockchain),
		}

	default:
		return fmt.Errorf(verr.Tag())
	}
}

// networkValidator verifies that the request is valid for the configured network.
// Network and blockchain fields must be populated and valid for the current
// running instance of the DPS. For example, a request cannot be addressing a Testnet
// network while we are reunning on top of a Mainnet index.
func (v *Validator) networkValidator(sl validator.StructLevel) {
	network := sl.Current().Interface().(identifier.Network)

	if network.Blockchain == "" {
		sl.ReportError(network.Blockchain, blockchainField, blockchainField, rosetta.BlockchainEmpty, "")
	}
	if network.Network == "" {
		sl.ReportError(network.Network, networkField, networkField, rosetta.NetworkEmpty, "")
	}

	err := v.config.Check(network)
	// Check returns a typed error, which we can use to identify which field was invalid.
	// We will use `tag` field to communicate this, and the `param` field to pass the acceptable
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

// blockValidator ensures that, if the block hash is provided, it has the correct length.
func blockValidator(sl validator.StructLevel) {
	rosBlockID := sl.Current().Interface().(identifier.Block)
	if rosBlockID.Hash != "" && len(rosBlockID.Hash) != rosetta.HexIDSize {
		sl.ReportError(rosBlockID.Hash, blockHashField, blockHashField, rosetta.BlockLength, "")
	}
}

// accountValidator ensures that the account address field is populated and has correct length.
func accountValidator(sl validator.StructLevel) {
	rosAccountID := sl.Current().Interface().(identifier.Account)
	if rosAccountID.Address == "" {
		sl.ReportError(rosAccountID.Address, addressField, addressField, rosetta.AddressEmpty, "")
	}
	if len(rosAccountID.Address) != rosetta.HexAddressSize {
		sl.ReportError(rosAccountID.Address, addressField, addressField, rosetta.AddressLength, "")
	}
}

// transactionValidator ensures that the transaction identifier is populated and has correct length.
func transactionValidator(sl validator.StructLevel) {
	rosTxID := sl.Current().Interface().(identifier.Transaction)
	if rosTxID.Hash == "" {
		sl.ReportError(rosTxID.Hash, txField, txField, rosetta.TxHashEmpty, "")
	}
	if len(rosTxID.Hash) != rosetta.HexIDSize {
		sl.ReportError(rosTxID.Hash, txField, txField, rosetta.TxLength, "")
	}
}

// balanceValidator ensures that the provided BalanceRequest has a non-empty currency list, and that
// all provided currencies have the `symbol` field populated.
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

// parseValidator ensures that the provided ParseRequest has a non-empty transaction field.
func parseValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(rosetta.ParseRequest)
	if req.Transaction == "" {
		sl.ReportError(req.Transaction, transactionField, transactionField, rosetta.TxBodyEmpty, "")
	}
}

// combineValidator ensures that the provided CombineRequest has a non-empty transaction field, and
// that the signature list is not empty.
func combineValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(rosetta.CombineRequest)
	if req.UnsignedTransaction == "" {
		sl.ReportError(req.UnsignedTransaction, transactionField, transactionField, rosetta.TxBodyEmpty, "")
	}

	if len(req.Signatures) == 0 {
		sl.ReportError(req.Signatures, signaturesField, signaturesField, rosetta.SignaturesEmpty, "")
	}
}

// submitValidator ensures that the provided SubmitRequest has a non-empty transaction field.
func submitValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(rosetta.SubmitRequest)
	if req.SignedTransaction == "" {
		sl.ReportError(req.SignedTransaction, transactionField, transactionField, rosetta.TxBodyEmpty, "")
	}
}

// hashValidator ensures that the provided SubmitRequest has a non-empty transaction field.
func hashValidator(sl validator.StructLevel) {
	req := sl.Current().Interface().(rosetta.HashRequest)
	if req.SignedTransaction == "" {
		sl.ReportError(req.SignedTransaction, transactionField, transactionField, rosetta.TxBodyEmpty, "")
	}
}
