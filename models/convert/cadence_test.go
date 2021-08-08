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

package convert

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence"

	"github.com/optakt/flow-dps/models/dps"
)

func TestParseCadenceArgument(t *testing.T) {

	tests := []struct {
		name     string
		param    string
		wantArg  cadence.Value
		checkErr assert.ErrorAssertionFunc
	}{
		{
			name:     "parse valid boolean",
			param:    "Bool(true)",
			wantArg:  cadence.Bool(true),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid boolean",
			param:    "Bool(horse)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid normal integer",
			param:    "Int16(1337)",
			wantArg:  cadence.Int16(1337),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid normal integer",
			param:    "Int16(a337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse invalid unsigned integer",
			param:    "UInt64(-1337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid big integer",
			param:    "UInt256(1337)",
			wantArg:  cadence.UInt256{Value: big.NewInt(1337)},
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid big integer",
			param:    "UInt128(a337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid fixed point",
			param:    "UFix64(13.37)",
			wantArg:  cadence.UFix64(1337000000),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid fixed point",
			param:    "Fix64(13,37)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid address",
			param:    "Address(43AC64656E636521)",
			wantArg:  cadence.Address{0x43, 0xac, 0x64, 0x65, 0x6e, 0x63, 0x65, 0x21},
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid address",
			param:    "Address(X3AC64656E636521)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid bytes",
			param:    "Bytes(43AC64656E636521)",
			wantArg:  cadence.Bytes{0x43, 0xac, 0x64, 0x65, 0x6e, 0x63, 0x65, 0x21},
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid bytes",
			param:    "Bytes(X3AC64656E636521)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid string",
			param:    "String(MN7wrJh359Kx+J*#)",
			wantArg:  cadence.String("MN7wrJh359Kx+J*#"),
			checkErr: assert.NoError,
		},
		{
			name:     "unsupported type",
			param:    "Doughnut(vanilla)",
			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotArg, err := ParseCadenceArgument(test.param)
			test.checkErr(t, err)

			if err == nil {
				assert.Equal(t, test.wantArg, gotArg)
			}
		})
	}
}

func TestParseRosettaValue(t *testing.T) {

	tests := []struct {
		name        string
		value       string
		fractionLen uint
		wantValue   cadence.UFix64
		checkErr    assert.ErrorAssertionFunc
	}{
		{
			name:        "parse 1.0",
			value:       "100000000",
			fractionLen: dps.FlowDecimals,
			wantValue:   100000000,
			checkErr:    assert.NoError,
		},
		{
			name:        "parse 12.34",
			value:       "1234000000",
			fractionLen: dps.FlowDecimals,
			wantValue:   1234000000,
			checkErr:    assert.NoError,
		},
		{
			name:        "parse 0.123456789",
			value:       "0123456789",
			fractionLen: dps.FlowDecimals,
			wantValue:   123456789,
			checkErr:    assert.NoError,
		},
		{
			name:        "parse 98",
			value:       "9800000000",
			fractionLen: dps.FlowDecimals,
			wantValue:   9800000000,
			checkErr:    assert.NoError,
		},
		{
			name:        "parse 0.00000001",
			value:       "000000001",
			fractionLen: dps.FlowDecimals,
			wantValue:   1,
			checkErr:    assert.NoError,
		},
		{
			name:        "parse 0.00000012",
			value:       "000000012",
			fractionLen: dps.FlowDecimals,
			wantValue:   12,
			checkErr:    assert.NoError,
		},
		{
			name:        "parse 1.000001",
			value:       "100000001",
			fractionLen: dps.FlowDecimals,
			wantValue:   100000001,
			checkErr:    assert.NoError,
		},
		{
			name:        "parse short value",
			value:       "1",
			fractionLen: dps.FlowDecimals,
			wantValue:   1,
			checkErr:    assert.NoError,
		},
		{
			name:        "parse value with invalid leading character",
			value:       "a123456789",
			fractionLen: dps.FlowDecimals,
			checkErr:    assert.Error,
		},
		{
			name:        "parse value with invalid trailing character",
			value:       "123456789a",
			fractionLen: dps.FlowDecimals,
			checkErr:    assert.Error,
		},
		{
			name:        "invalid number of decimals",
			value:       "123456789",
			fractionLen: 7,
			checkErr:    assert.Error,
		},
		{
			name:        "parse negative value",
			value:       "-123456789",
			fractionLen: dps.FlowDecimals,
			checkErr:    assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			v, err := ParseRosettaValue(test.value, test.fractionLen)
			test.checkErr(t, err)
			assert.Equal(t, test.wantValue, v)
		})
	}
}
