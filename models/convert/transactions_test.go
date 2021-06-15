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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

func TestEventsToTransactions(t *testing.T) {
	depositType := &cadence.EventType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "deposit",
		Fields: []cadence.Field{
			{
				Identifier: "amount",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "address",
				Type:       cadence.AddressType{},
			},
		},
	}
	depositEvent := cadence.NewEvent(
		[]cadence.Value{
			cadence.NewUInt64(42),
			cadence.NewAddress([8]byte{1, 2, 3, 4, 5, 6, 7, 8}),
		},
	).WithType(depositType)
	depositEventPayload := json.MustEncode(depositEvent)

	withdrawalType := &cadence.EventType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "withdrawal",
		Fields: []cadence.Field{
			{
				Identifier: "amount",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "address",
				Type:       cadence.AddressType{},
			},
		},
	}
	withdrawalEvent := cadence.NewEvent(
		[]cadence.Value{
			cadence.NewUInt64(42),
			cadence.NewAddress([8]byte{2, 3, 4, 5, 6, 7, 8, 9}),
		},
	).WithType(withdrawalType)
	withdrawalEventPayload := json.MustEncode(withdrawalEvent)

	testDepositOp1 := object.Operation{
		ID: identifier.Operation{
			Index: 0,
		},
		RelatedIDs: []identifier.Operation{
			{Index: 1},
		},
		Type:   dps.OperationTransfer,
		Status: dps.StatusCompleted,
		AccountID: identifier.Account{
			Address: "0102030405060708",
		},
		Amount: object.Amount{
			Value: "42",
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}
	testWithdrawalOp1 := object.Operation{
		ID: identifier.Operation{
			Index: 1,
		},
		RelatedIDs: []identifier.Operation{
			{Index: 0},
		},
		Type:   dps.OperationTransfer,
		Status: dps.StatusCompleted,
		AccountID: identifier.Account{
			Address: "0203040506070809",
		},
		Amount: object.Amount{
			Value: "-42",
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}
	testDepositOp2 := object.Operation{
		ID: identifier.Operation{
			Index: 2,
		},
		RelatedIDs: []identifier.Operation{
			{Index: 3},
		},
		Type:   dps.OperationTransfer,
		Status: dps.StatusCompleted,
		AccountID: identifier.Account{
			Address: "0102030405060708",
		},
		Amount: object.Amount{
			Value: "42",
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}
	testWithdrawalOp2 := object.Operation{
		ID: identifier.Operation{
			Index: 3,
		},
		RelatedIDs: []identifier.Operation{
			{Index: 2},
		},
		Type:   dps.OperationTransfer,
		Status: dps.StatusCompleted,
		AccountID: identifier.Account{
			Address: "0203040506070809",
		},
		Amount: object.Amount{
			Value: "-42",
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}

	id1, err := flow.HexStringToIdentifier("a4c4194eae1a2dd0de4f4d51a884db4255bf265a40ddd98477a1d60ef45909ec")
	require.NoError(t, err)

	id2, err := flow.HexStringToIdentifier("e956563a78d74f927ccf1e81b4c3a012e691e9c30393bcfdb8b3db5060d4075b")
	require.NoError(t, err)

	testTransaction1 := &object.Transaction{
		ID: identifier.Transaction{
			Hash: id1.String(),
		},
		Operations: []object.Operation{
			testDepositOp1,
			testWithdrawalOp1,
		},
	}
	testTransaction2 := &object.Transaction{
		ID: identifier.Transaction{
			Hash: id2.String(),
		},
		Operations: []object.Operation{
			testDepositOp2,
			testWithdrawalOp2,
		},
	}

	threeFieldsType := &cadence.EventType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "test",
		Fields: []cadence.Field{
			{
				Identifier: "testField1",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "testField2",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "testField3",
				Type:       cadence.UInt64Type{},
			},
		},
	}
	threeFieldsEvent := cadence.NewEvent(
		[]cadence.Value{
			cadence.NewUInt64(42),
			cadence.NewUInt64(42),
			cadence.NewUInt64(42),
		},
	).WithType(threeFieldsType)
	threeFieldsEventPayload := json.MustEncode(threeFieldsEvent)

	missingAmountEventType := &cadence.EventType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "test",
		Fields: []cadence.Field{
			{
				Identifier: "address",
				Type:       cadence.AddressType{},
			},
			{
				Identifier: "testField",
				Type:       cadence.AddressType{},
			},
		},
	}
	missingAmountEvent := cadence.NewEvent(
		[]cadence.Value{
			cadence.NewAddress([8]byte{1, 2, 3, 4, 5, 6, 7, 8}),
			cadence.NewAddress([8]byte{1, 2, 3, 4, 5, 6, 7, 8}),
		},
	).WithType(missingAmountEventType)
	missingAmountEventPayload := json.MustEncode(missingAmountEvent)

	missingAddressEventType := &cadence.EventType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "test",
		Fields: []cadence.Field{
			{
				Identifier: "amount",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "amount",
				Type:       cadence.UInt64Type{},
			},
		},
	}
	missingAddressEvent := cadence.NewEvent(
		[]cadence.Value{
			cadence.NewUInt64(42),
			cadence.NewUInt64(42),
		},
	).WithType(missingAddressEventType)
	missingAddressEventPayload := json.MustEncode(missingAddressEvent)

	tests := []struct {
		description string

		events []flow.Event

		wantTransactions []*object.Transaction
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			description: "nominal case with one single transaction",

			events: []flow.Event{{
				TransactionID: id1,
				Type:          "deposit",
				Payload:       depositEventPayload,
				EventIndex:    0,
			}, {
				TransactionID: id1,
				Type:          "withdrawal",
				Payload:       withdrawalEventPayload,
				EventIndex:    1,
			}},

			wantErr: assert.NoError,
			wantTransactions: []*object.Transaction{
				testTransaction1,
			},
		},
		{
			description: "nominal case with multiple transactions",

			events: []flow.Event{
				{
					TransactionID: id1,
					Type:          "deposit",
					Payload:       depositEventPayload,
					EventIndex:    0,
				},
				{
					TransactionID: id1,
					Type:          "withdrawal",
					Payload:       withdrawalEventPayload,
					EventIndex:    1,
				},
				{
					TransactionID: id2,
					Type:          "deposit",
					Payload:       depositEventPayload,
					EventIndex:    2,
				},
				{
					TransactionID: id2,
					Type:          "withdrawal",
					Payload:       withdrawalEventPayload,
					EventIndex:    3,
				},
			},

			wantErr: assert.NoError,
			wantTransactions: []*object.Transaction{
				testTransaction1,
				testTransaction2,
			},
		},
		{
			description: "wrong amount of fields",

			events: []flow.Event{{
				Payload: threeFieldsEventPayload,
			}},

			wantErr: assert.Error,
		},
		{
			description: "missing amount field",

			events: []flow.Event{{
				Payload: missingAmountEventPayload,
			}},

			wantErr: assert.Error,
		},
		{
			description: "missing address field",

			events: []flow.Event{{
				Payload: missingAddressEventPayload,
			}},

			wantErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			got, err := EventsToTransactions(test.events, "withdrawal")

			test.wantErr(t, err)

			// Since the result is a map, use assert.Contains in order not to rely on order.
			for _, gotTransaction := range got {
				assert.Contains(t, test.wantTransactions, gotTransaction)
			}
		})
	}
}
