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

package storage

import (
	"encoding/binary"
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

func EncodeKey(prefix uint8, segments ...interface{}) []byte {
	key := []byte{prefix}
	var val []byte
	for _, segment := range segments {
		switch s := segment.(type) {
		case uint64:
			val = make([]byte, 8)
			binary.BigEndian.PutUint64(val, s)
		case flow.Identifier:
			val = make([]byte, 32)
			copy(val, s[:])
		case ledger.Path:
			val = make([]byte, 32)
			copy(val, s[:])
		case flow.StateCommitment:
			val = make([]byte, 32)
			copy(val, s[:])
		default:
			panic(fmt.Sprintf("unknown type (%T)", segment))
		}
		key = append(key, val...)
	}

	return key
}
