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

package convert_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence"

	"github.com/onflow/flow-dps/models/convert"
)

func TestParseCadenceArgument(t *testing.T) {
	tests := []struct {
		name     string
		param    string
		wantArg  cadence.Value
		checkErr assert.ErrorAssertionFunc
	}{
		{
			name:     "handles invalid parameter format",
			param:    "test",
			checkErr: assert.Error,
		},
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
			param:    "Int(1337)",
			wantArg:  cadence.Int{Value: big.NewInt(1337)},
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid normal integer",
			param:    "Int(a337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid short integer",
			param:    "Int8(127)",
			wantArg:  cadence.Int8(127),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid short integer",
			param:    "Int8(a27)",
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
			name:     "parse valid 32-bit integer",
			param:    "Int32(1337)",
			wantArg:  cadence.Int32(1337),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid 32-bit integer",
			param:    "Int32(a337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid 64-bit integer",
			param:    "Int64(1337)",
			wantArg:  cadence.Int64(1337),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid 64-bit integer",
			param:    "Int64(a337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid 128-bit integer",
			param:    "Int128(1337)",
			wantArg:  cadence.Int128{Value: big.NewInt(1337)},
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid 64-bit integer",
			param:    "Int128(a337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid 256-bit integer",
			param:    "Int256(1337)",
			wantArg:  cadence.Int256{Value: big.NewInt(1337)},
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid 256-bit integer",
			param:    "Int256(a337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid unsigned integer",
			param:    "UInt(1337)",
			wantArg:  cadence.UInt{Value: big.NewInt(1337)},
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid unsigned integer",
			param:    "UInt(-1337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid short unsigned integer",
			param:    "UInt8(127)",
			wantArg:  cadence.UInt8(127),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid short unsigned integer",
			param:    "UInt8(-127)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid 16-bit unsigned integer",
			param:    "UInt16(1337)",
			wantArg:  cadence.UInt16(1337),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid 16-bit unsigned integer",
			param:    "UInt16(-1337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid 32-bit unsigned integer",
			param:    "UInt32(1337)",
			wantArg:  cadence.UInt32(1337),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid 32-bit unsigned integer",
			param:    "UInt32(-1337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid 64-bit unsigned integer",
			param:    "UInt64(1337)",
			wantArg:  cadence.UInt64(1337),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid 64-bit unsigned integer",
			param:    "UInt64(-1337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid 128-bit unsigned integer",
			wantArg:  cadence.UInt128{Value: big.NewInt(1337)},
			param:    "UInt128(1337)",
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid 128-bit unsigned integer",
			param:    "UInt128(-1337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid big unsigned integer",
			param:    "UInt256(1337)",
			wantArg:  cadence.UInt256{Value: big.NewInt(1337)},
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid big unsigned integer",
			param:    "UInt256(-1337)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid unsigned fixed point",
			param:    "UFix64(13.37)",
			wantArg:  cadence.UFix64(1337000000),
			checkErr: assert.NoError,
		},
		{
			name:     "parse invalid unsigned fixed point",
			param:    "UFix64(13,37)",
			checkErr: assert.Error,
		},
		{
			name:     "parse valid fixed point",
			param:    "Fix64(13.37)",
			wantArg:  cadence.Fix64(1337000000),
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

			gotArg, err := convert.ParseCadenceArgument(test.param)
			test.checkErr(t, err)

			if err == nil {
				assert.Equal(t, test.wantArg, gotArg)
			}
		})
	}
}
