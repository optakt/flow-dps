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

package zbor

import (
	"encoding/hex"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/dictionaries"
)

type Codec struct {
	encMode cbor.EncMode

	defaultCompressor *zstd.Encoder
	headerCompressor  *zstd.Encoder
	payloadCompressor *zstd.Encoder
	eventsCompressor  *zstd.Encoder

	decompressor *zstd.Decoder
}

// NewCodec creates a new Codec.
func NewCodec() (*Codec, error) {
	codec, _ := dps.Encoding.EncMode()

	defaultCompressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize default compressor: %w", err)
	}

	headerDict, err := hex.DecodeString(dictionaries.Header)
	if err != nil {
		return nil, fmt.Errorf("could not decode header dictionary: %w", err)
	}

	headerCompressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(headerDict),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize header compressor: %w", err)
	}

	payloadDict, err := hex.DecodeString(dictionaries.Payload)
	if err != nil {
		return nil, fmt.Errorf("could not decode payload dictionary: %w", err)
	}
	payloadCompressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(payloadDict),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize payload compressor: %w", err)
	}

	eventsDict, err := hex.DecodeString(dictionaries.Events)
	if err != nil {
		return nil, fmt.Errorf("could not decode events dictionary: %w", err)
	}

	eventsCompressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(eventsDict),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize events compressor: %w", err)
	}

	decompressor, err := zstd.NewReader(nil,
		zstd.WithDecoderDicts(headerDict, payloadDict, eventsDict),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize decompressor: %w", err)
	}

	c := Codec{
		encMode: codec,

		defaultCompressor: defaultCompressor,
		headerCompressor:  headerCompressor,
		eventsCompressor:  eventsCompressor,
		payloadCompressor: payloadCompressor,

		decompressor: decompressor,
	}

	return &c, nil
}

func (c *Codec) Unmarshal(b []byte, value interface{}) error {
	val, err := c.decompressor.DecodeAll(b, nil)
	if err != nil {
		return fmt.Errorf("unable to decompress value: %w", err)
	}

	err = cbor.Unmarshal(val, value)
	if err != nil {
		return fmt.Errorf("unable to decode value: %w", err)
	}

	return nil
}

func (c *Codec) Marshal(v interface{}) ([]byte, error) {
	b, err := cbor.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("unable to encode value: %w", err)
	}

	compressor := c.defaultCompressor
	switch v.(type) {
	case *flow.Header:
		compressor = c.headerCompressor
	case *ledger.Payload:
		compressor = c.payloadCompressor
	case []flow.Event:
		compressor = c.eventsCompressor
	}

	return compressor.EncodeAll(b, nil), nil
}
