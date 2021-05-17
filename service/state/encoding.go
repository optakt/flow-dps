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

package state

import "encoding/binary"

func Encode(pins ...interface{}) []byte {
	var key []byte
	var it int
	for _, pin := range pins {
		switch pin.(type) {
		case uint8:
			key = append(key, pin.(uint8))
			it += 1
		case uint64:
			val := make([]byte, 8)

			binary.BigEndian.PutUint64(val, pin.(uint64))
			key = append(key, val...)
			it += 8
		case []byte:
			payload := pin.([]byte)
			val := make([]byte, len(payload))

			copy(val, payload)
			key = append(key, val...)
			it += len(payload)
		default:
			panic("unknown type")
		}
	}

	return key
}

func Decode(key []byte, pins ...interface{}) error {
	var it int
	for _, pin := range pins {
		switch pin.(type) {
		case *uint64:
			ptr := pin.(*uint64)
			*ptr = binary.BigEndian.Uint64(key[it:it+8])
			it += 8
		case *[]byte:
			ptr := pin.(*[]byte)
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
			panic("unknown type")
		}
	}

	return nil
}
