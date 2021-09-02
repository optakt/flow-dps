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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/ledger"

	"github.com/optakt/flow-dps/models/convert"
)

func TestValuesToBytes(t *testing.T) {
	val1b := []byte{0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70, 0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1}
	val2b := []byte{0x14, 0x54, 0xae, 0x24, 0x20, 0x51, 0x3f, 0x79, 0xf6, 0xa5, 0xe8, 0x39, 0x6d, 0x03, 0x33, 0x69}
	val3b := []byte{0xe9, 0x1a, 0x3e, 0xb9, 0x97, 0x75, 0x2b, 0x78, 0xba, 0xb0, 0xbc, 0x31, 0xe3, 0x0b, 0x1e, 0x30}

	val1 := ledger.Value(val1b)
	val2 := ledger.Value(val2b)
	val3 := ledger.Value(val3b)

	vals := []ledger.Value{val1, val2, val3}

	got := convert.ValuesToBytes(vals)

	assert.Equal(t, [][]byte{val1b, val2b, val3b}, got)
}

func TestBytesToValues(t *testing.T) {
	val1b := []byte{0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70, 0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1}
	val2b := []byte{0x14, 0x54, 0xae, 0x24, 0x20, 0x51, 0x3f, 0x79, 0xf6, 0xa5, 0xe8, 0x39, 0x6d, 0x03, 0x33, 0x69}
	val3b := []byte{0xe9, 0x1a, 0x3e, 0xb9, 0x97, 0x75, 0x2b, 0x78, 0xba, 0xb0, 0xbc, 0x31, 0xe3, 0x0b, 0x1e, 0x30}

	val1 := ledger.Value(val1b)
	val2 := ledger.Value(val2b)
	val3 := ledger.Value(val3b)

	wantVals := []ledger.Value{val1, val2, val3}

	bb := [][]byte{val1b, val2b, val3b}

	got := convert.BytesToValues(bb)

	assert.Equal(t, wantVals, got)
}
