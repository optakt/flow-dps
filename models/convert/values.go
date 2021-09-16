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
	"github.com/onflow/flow-go/ledger"
)

// ValuesToBytes converts a slice of ledger values into a slice of byte slices.
func ValuesToBytes(values []ledger.Value) [][]byte {
	bb := make([][]byte, 0, len(values))
	for _, value := range values {
		b := make([]byte, len(value))
		copy(b, value[:])
		bb = append(bb, b)
	}
	return bb
}

// BytesToValues converts a slice of byte slices into a slice of ledger values.
func BytesToValues(bb [][]byte) []ledger.Value {
	values := make([]ledger.Value, 0, len(bb))
	for _, b := range bb {
		value := ledger.Value(b)
		values = append(values, value)
	}
	return values
}
