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

	"github.com/onflow/flow-go/ledger"
)

func TestValuesToBytes(t *testing.T) {
	val1b := []byte("aac513eb1a0457700ac3fa8d292513e1")
	val2b := []byte("1454ae2420513f79f6a5e8396d033369")
	val3b := []byte("e91a3eb997752b78bab0bc31e30b1e30")

	val1 := ledger.Value(val1b)
	val2 := ledger.Value(val2b)
	val3 := ledger.Value(val3b)

	vals := []ledger.Value{val1, val2, val3}

	got := ValuesToBytes(vals)

	assert.Equal(t, [][]byte{val1b, val2b, val3b}, got)
}

func TestBytesToValues(t *testing.T) {
	val1b := []byte("aac513eb1a0457700ac3fa8d292513e1")
	val2b := []byte("1454ae2420513f79f6a5e8396d033369")
	val3b := []byte("e91a3eb997752b78bab0bc31e30b1e30")

	val1 := ledger.Value(val1b)
	val2 := ledger.Value(val2b)
	val3 := ledger.Value(val3b)

	wantVals := []ledger.Value{val1, val2, val3}

	bb := [][]byte{val1b, val2b, val3b}

	got, err := BytesToValues(bb)

	assert.NoError(t, err)
	assert.Equal(t, wantVals, got)
}
