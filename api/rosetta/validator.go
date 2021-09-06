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
	// Special case in the validator library that shouldn't really happen with correct usage.
	errInvalidValidation = errors.New("invalid validation")

	errBlockLength   = errors.New(blockLength)
	errAddressLength = errors.New(addressLength)
	errTxLength      = errors.New(txLength)
)

// PR comment - this could be unexported, but then there are name conflicts.
// I didn't come up with a nice name which didn't lead to N instances of validate/validator.
type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {

	v := validator.New()

	// Register custom validators for known types.
	v.RegisterStructValidation(blockValidator, identifier.Block{})
	v.RegisterStructValidation(networkValidator, identifier.Network{})
	v.RegisterStructValidation(accountValidator, identifier.Account{})
	v.RegisterStructValidation(transactionValidator, identifier.Transaction{})
	v.RegisterStructValidation(currencyValidator, identifier.Currency{})

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

	// PR comment - this only happens if an invalid argument was given to the struct validation
	// functions. Still, since we're doing type assertsions, I thought it's safer to check it here
	// and ignore on the callers side.
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
