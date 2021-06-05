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
	"encoding/hex"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/dictionaries"
)

// TODO: Extract all encoding/decoding and compression code into a single
// component that can be used across all other components (i.e. GRPC API,
// storage, live mapper pub/sub and req/rep sockets).
// => https://github.com/optakt/flow-dps/issues/120

var (
	codec             cbor.EncMode
	defaultCompressor *zstd.Encoder
	headerCompressor  *zstd.Encoder
	payloadCompressor *zstd.Encoder
	eventsCompressor  *zstd.Encoder
	decompressor      *zstd.Decoder
)

func init() {
	var err error

	codec, _ = dps.Encoding.EncMode()

	defaultCompressor, err = zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
	)
	if err != nil {
		panic(fmt.Errorf("could not initialize default compressor: %w", err))
	}

	headerDict, err := hex.DecodeString(dictionaries.Header)
	if err != nil {
		panic(fmt.Errorf("could not decode header dictionary: %w", err))
	}

	headerCompressor, err = zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(headerDict),
	)
	if err != nil {
		panic(fmt.Errorf("could not initialize header compressor: %w", err))
	}

	payloadDict, err := hex.DecodeString(dictionaries.Payload)
	if err != nil {
		panic(fmt.Errorf("could not decode payload dictionary: %w", err))
	}

	payloadCompressor, err = zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(payloadDict),
	)
	if err != nil {
		panic(fmt.Errorf("could not initialize payload compressor: %w", err))
	}

	eventsDict, err := hex.DecodeString(dictionaries.Events)
	if err != nil {
		panic(fmt.Errorf("could not decode events dictionary: %w", err))
	}

	eventsCompressor, err = zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(eventsDict),
	)
	if err != nil {
		panic(fmt.Errorf("could not initialize events compressor: %w", err))
	}

	decompressor, err = zstd.NewReader(nil,
		zstd.WithDecoderDicts(headerDict, payloadDict, eventsDict),
	)
	if err != nil {
		panic(fmt.Errorf("could not initialize decompressor: %w", err))
	}
}

func encodeKey(prefix uint8, segments ...interface{}) []byte {
	key := []byte{prefix}
	var val []byte
	for _, segment := range segments {
		switch s := segment.(type) {
		case uint64:
			val = make([]byte, 8)
			binary.BigEndian.PutUint64(val, s)
		case flow.Identifier:
			val = s[:]
		case ledger.Path:
			val = s[:]
		case flow.StateCommitment:
			val = s[:]
		default:
			panic(fmt.Sprintf("unknown type (%T)", segment))
		}
		key = append(key, val...)
	}

	return key
}
