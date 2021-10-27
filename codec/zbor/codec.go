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
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

// Codec encodes and decodes Go values using cbor encoding and zstandard compression.
type Codec struct {
	encoder cbor.EncMode
	decoder cbor.DecMode

	compressor              *zstd.Encoder
	decompressor            *zstd.Decoder
	payloadCompressor       *zstd.Encoder
	payloadDecompressor     *zstd.Decoder
	eventCompressor         *zstd.Encoder
	eventDecompressor       *zstd.Decoder
	transactionCompressor   *zstd.Encoder
	transactionDecompressor *zstd.Decoder
}

// NewCodec creates a new Codec.
func NewCodec() *Codec {

	// We should never fail here if the options are valid, so use panic to keep
	// the function signature for the codec clean.
	encOptions := cbor.CanonicalEncOptions()
	encOptions.Time = cbor.TimeRFC3339Nano
	encoder, err := encOptions.EncMode()
	if err != nil {
		panic(err)
	}

	decOptions := cbor.DecOptions{
		ExtraReturnErrors: cbor.ExtraDecErrorUnknownField,
	}
	decoder, err := decOptions.DecMode()
	if err != nil {
		panic(err)
	}

	compressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(genericDictionary),
	)
	if err != nil {
		panic(err)
	}
	decompressor, err := zstd.NewReader(nil,
		zstd.WithDecoderDicts(
			genericDictionary,
			legacyGenericDictionary,
			legacyHeaderDictionary,
		),
	)
	if err != nil {
		panic(err)
	}

	payloadCompressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(payloadDictionary),
	)
	if err != nil {
		panic(err)
	}
	payloadDecompressor, err := zstd.NewReader(nil,
		zstd.WithDecoderDicts(
			payloadDictionary,
			legacyPayloadDictionary,
		),
	)
	if err != nil {
		panic(err)
	}

	eventCompressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(eventDictionary),
	)
	if err != nil {
		panic(err)
	}
	eventDecompressor, err := zstd.NewReader(nil,
		zstd.WithDecoderDicts(
			eventDictionary,
			legacyEventDictionary,
		),
	)
	if err != nil {
		panic(err)
	}

	transactionCompressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(transactionDictionary),
	)
	if err != nil {
		panic(err)
	}
	transactionDecompressor, err := zstd.NewReader(nil,
		zstd.WithDecoderDicts(transactionDictionary),
	)
	if err != nil {
		panic(err)
	}

	c := Codec{
		encoder: encoder,
		decoder: decoder,

		compressor:              compressor,
		decompressor:            decompressor,
		payloadCompressor:       payloadCompressor,
		payloadDecompressor:     payloadDecompressor,
		eventCompressor:         eventCompressor,
		eventDecompressor:       eventDecompressor,
		transactionCompressor:   transactionCompressor,
		transactionDecompressor: transactionDecompressor,
	}

	return &c
}

// Encode returns the CBOR encoding of the given value.
func (c *Codec) Encode(value interface{}) ([]byte, error) {
	return c.encoder.Marshal(value)
}

// Compress encodes the given bytes into a compressed format using zstandard.
func (c *Codec) Compress(data []byte) ([]byte, error) {
	compressed := c.compressor.EncodeAll(data, nil)
	return compressed, nil
}

// Marshal encodes the given value and then compresses it, and returns the resulting slice of bytes.
func (c *Codec) Marshal(value interface{}) ([]byte, error) {
	data, err := c.Encode(value)
	if err != nil {
		return nil, fmt.Errorf("could not encode value: %w", err)
	}

	var compressed []byte
	switch value.(type) {
	case *ledger.Payload:
		compressed = c.payloadCompressor.EncodeAll(data, nil)
	case []flow.Event:
		compressed = c.eventCompressor.EncodeAll(data, nil)
	case *flow.TransactionBody:
		compressed = c.transactionCompressor.EncodeAll(data, nil)
	default:
		compressed = c.compressor.EncodeAll(data, nil)
	}

	return compressed, nil
}

// Decode parses CBOR-encoded data into the given value.
func (c *Codec) Decode(data []byte, value interface{}) error {
	return c.decoder.Unmarshal(data, value)
}

// Decompress reads compressed data that uses the zstandard format and returns the original
// uncompressed byte slice.
func (c *Codec) Decompress(compressed []byte) ([]byte, error) {
	data, err := c.decompressor.DecodeAll(compressed, nil)
	return data, err
}

// Unmarshal decompresses the given bytes and decodes the resulting CBOR-encoded data into
// the given value.
func (c *Codec) Unmarshal(compressed []byte, value interface{}) error {

	var data []byte
	var err error
	switch value.(type) {
	case *ledger.Payload:
		data, err = c.payloadDecompressor.DecodeAll(compressed, nil)
	case *[]flow.Event:
		data, err = c.eventDecompressor.DecodeAll(compressed, nil)
	case *flow.TransactionBody:
		data, err = c.transactionDecompressor.DecodeAll(compressed, nil)
	default:
		data, err = c.decompressor.DecodeAll(compressed, nil)
	}
	if err != nil {
		return fmt.Errorf("could not decompress value: %w", err)
	}

	err = c.Decode(data, value)
	if err != nil {
		return fmt.Errorf("could not decode value: %w", err)
	}
	return nil
}
