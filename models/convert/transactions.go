// Copyright 2021 Alvalor S.A.
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

package convert

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

// EventsToTransactions processes a slice of events into a map of Rosetta transactions. It ensures that
// each operation included in transactions has its related operation IDs set accordingly, and maps transactions
// by their ID.
func EventsToTransactions(ee []flow.Event, withdrawal string) (map[string]*object.Transaction, error) {
	transactions := make(map[string]*object.Transaction)
	for _, event := range ee {
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

		// In the case of a withdrawal, invert the amount value.
		if event.Type == flow.EventType(withdrawal) {
			amount = -amount
		}

		op := object.Operation{
			ID: identifier.Operation{
				Index: uint(event.EventIndex),
			},
			RelatedIDs: nil,
			Type:       "TRANSFER",
			Status:     "COMPLETED",
			AccountID: identifier.Account{
				Address: address.String(),
			},
			Amount: object.Amount{
				Value: strconv.FormatInt(amount, 10),
				Currency: identifier.Currency{
					Symbol:   dps.FlowSymbol,
					Decimals: dps.FlowDecimals,
				},
			},
		}

		transaction, exists := transactions[event.TransactionID.String()]
		if !exists {
			transaction = &object.Transaction{
				ID: identifier.Transaction{
					Hash: event.TransactionID.String(),
				},
			}
			transactions[event.TransactionID.String()] = transaction
		}

		transaction.Operations = append(transactions[event.TransactionID.String()].Operations, op)
	}

	// Go through all operations of each transaction and set their related IDs.
	for tIdx := range transactions {
		for oIdx, op1 := range transactions[tIdx].Operations {
			for _, op2 := range transactions[tIdx].Operations {
				if op1.ID == op2.ID {
					continue
				}

				transactions[tIdx].Operations[oIdx].RelatedIDs = append(transactions[tIdx].Operations[oIdx].RelatedIDs, op2.ID)
			}
		}
	}

	return transactions, nil
}
