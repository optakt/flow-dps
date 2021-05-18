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

package storage

import (
	"encoding/binary"
	"fmt"
)

func Encode(prefix uint8, segments ...interface{}) []byte {
	key := []byte{prefix}
	var it int
	for _, segment := range segments {
		switch s := segment.(type) {
		case uint64:
			val := make([]byte, 8)

			binary.BigEndian.PutUint64(val, s)
			key = append(key, val...)
			it += 8
		case []byte:
			val := make([]byte, len(s))

			copy(val, s)
			key = append(key, val...)
			it += len(s)
		default:
			panic("unknown type")
		}
	}

	return key
}

func Decode(key []byte, segments ...interface{}) (prefix uint8, err error) {
	var it int

	if len(key) > 0 {
		prefix = key[0]
		it++
	}
	for _, segment := range segments {
		switch ptr := segment.(type) {
		case *uint64:
			*ptr = binary.BigEndian.Uint64(key[it : it+8])
			it += 8
		case *[]byte:
			length := len(*ptr)
			if length == 0 { // This makes it possible to skip a pin by just giving nil.
				continue
			}

			// Retrieve value.
			val := make([]byte, length)
			copy(val, key[it:it+length])

			*ptr = val
			it += length
		default:
			return prefix, fmt.Errorf("unknown segment type %T", ptr)
		}
	}

	return prefix, nil
}
