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

package converter

import (
	"fmt"
	"strconv"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type Converter struct {
	deposit    flow.EventType
	withdrawal flow.EventType
}

func New(gen Generator) (*Converter, error) {
	deposit, err := gen.TokensDeposited(dps.FlowSymbol)
	if err != nil {
		return nil, fmt.Errorf("could not generate deposit event type: %w", err)
	}
	withdrawal, err := gen.TokensWithdrawn(dps.FlowSymbol)
	if err != nil {
		return nil, fmt.Errorf("could not generate withdrawal event type: %w", err)
	}

	// TODO: Generate more event types (stake/unstake, delegate/lock, etc.)
	//		 https://github.com/optakt/flow-dps/issues/214

	c := Converter{
		deposit:    flow.EventType(deposit),
		withdrawal: flow.EventType(withdrawal),
	}

	return &c, nil
}

func (c *Converter) EventToOperation(index uint, event flow.Event) (operation *object.Operation, err error) {

	// Decode the event payload into a Cadence value and cast it to a Cadence event.
	value, err := json.Decode(event.Payload)
	if err != nil {
		return nil, fmt.Errorf("could not decode event: %w", err)
	}
	e, ok := value.(cadence.Event)
	if !ok {
		return nil, fmt.Errorf("could not cast event: %w", err)
	}

	// Ensure that there are the correct amount of fields.
	if len(e.Fields) != 2 {
		return nil, fmt.Errorf("invalid number of fields (want: %d, have: %d)", 2, len(e.Fields))
	}

	// The first field is always the amount and the second one the address.
	// The types coming from Cadence are not native Flow types, so primitive types
	// are needed before they can be converted into proper Flow types.
	vAmount := e.Fields[0].ToGoValue()
	uAmount, ok := vAmount.(uint64)
	if !ok {
		return nil, fmt.Errorf("could not cast amount (%T)", vAmount)
	}
	vAddress := e.Fields[1].ToGoValue()
	bAddress, ok := vAddress.([flow.AddressLength]byte)
	if !ok {
		return nil, fmt.Errorf("could not cast address (%T)", vAddress)
	}

	// Convert the amount to a signed integer that it can be inverted.
	amount := int64(uAmount)
	// Convert the address bytes into a native Flow address.
	address := flow.Address(bAddress)

	op := object.Operation{
		ID: identifier.Operation{
			Index:        index,
			NetworkIndex: uint(event.EventIndex),
		},
		Status: dps.StatusCompleted,
		AccountID: identifier.Account{
			Address: address.String(),
		},
	}

	switch event.Type {
	// TODO: Support more event types (stake/unstake, delegate/lock, etc.)
	//		 https://github.com/optakt/flow-dps/issues/214
	case c.deposit:
		op.Type = dps.OperationTransfer

	// In the case of a withdrawal, invert the amount value.
	case c.withdrawal:
		op.Type = dps.OperationTransfer
		amount = -amount
	default:
		return nil, ErrIrrelevant
	}

	op.Amount = object.Amount{
		Value: strconv.FormatInt(amount, 10),
		Currency: identifier.Currency{
			Symbol:   dps.FlowSymbol,
			Decimals: dps.FlowDecimals,
		},
	}

	return &op, nil
}
