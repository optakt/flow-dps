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
	"math/big"
	"testing"

	"github.com/onflow/cadence"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCadenceArgument(t *testing.T) {

	vectors := []struct {
		name     string
		param    string
		wantArg  cadence.Value
		checkErr require.ErrorAssertionFunc
	}{
		{
			name:     "parse valid normal integer",
			param:    "Int16(1337)",
			wantArg:  cadence.Int16(1337),
			checkErr: require.NoError,
		},
		{
			name:     "parse valid big integer",
			param:    "UInt256(1337)",
			wantArg:  cadence.UInt256{Value: big.NewInt(1337)},
			checkErr: require.NoError,
		},
		{
			name:     "parse valid fixed point",
			param:    "UFix64(13.37)",
			wantArg:  cadence.UFix64(1337),
			checkErr: require.NoError,
		},
		{
			name:     "parse valid address",
			param:    "Address(43AC64656E636521)",
			wantArg:  cadence.Address{0x43, 0xac, 0x64, 0x65, 0x6e, 0x63, 0x65, 0x23},
			checkErr: require.NoError,
		},
		{
			name:     "parse valid bytes",
			param:    "Bytes(43AC64656E636521)",
			wantArg:  cadence.Bytes{0x43, 0xac, 0x64, 0x65, 0x6e, 0x63, 0x65, 0x23},
			checkErr: require.NoError,
		},
		{
			name:     "parse valid string",
			param:    "String(MN7wrJh359Kx+J*#)",
			wantArg:  cadence.String("MN7wrJh359Kx+J*#"),
			checkErr: require.NoError,
		},
	}

	for _, vector := range vectors {

		gotArg, err := ParseCadenceArgument(vector.param)
		vector.checkErr(t, err)

		if err != nil {
			assert.Equal(t, vector.wantArg, gotArg)
		}
	}
}
