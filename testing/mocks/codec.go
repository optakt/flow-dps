package mocks

import "testing"

type Codec struct {
	EncodeFunc     func(value interface{}) ([]byte, error)
	DecodeFunc     func(data []byte, value interface{}) error
	CompressFunc   func(data []byte) ([]byte, error)
	DecompressFunc func(compressed []byte) ([]byte, error)
	MarshalFunc    func(value interface{}) ([]byte, error)
	UnmarshalFunc  func(compressed []byte, value interface{}) error
}

func BaselineCodec(t *testing.T) *Codec {
	t.Helper()

	c := Codec{
		EncodeFunc: func(interface{}) ([]byte, error) {
			return GenericBytes, nil
		},
		DecodeFunc: func([]byte, interface{}) error {
			return nil
		},
		CompressFunc: func([]byte) ([]byte, error) {
			return GenericBytes, nil
		},
		DecompressFunc: func([]byte) ([]byte, error) {
			return GenericBytes, nil
		},
		UnmarshalFunc: func([]byte, interface{}) error {
			return nil
		},
		MarshalFunc: func(interface{}) ([]byte, error) {
			return GenericBytes, nil
		},
	}

	return &c
}

func (c *Codec) Encode(value interface{}) ([]byte, error) {
	return c.EncodeFunc(value)
}

func (c *Codec) Decode(data []byte, value interface{}) error {
	return c.DecodeFunc(data, value)
}

func (c *Codec) Compress(data []byte) ([]byte, error) {
	return c.CompressFunc(data)
}

func (c *Codec) Decompress(data []byte) ([]byte, error) {
	return c.DecompressFunc(data)
}

func (c *Codec) Unmarshal(b []byte, v interface{}) error {
	return c.UnmarshalFunc(b, v)
}

func (c *Codec) Marshal(v interface{}) ([]byte, error) {
	return c.MarshalFunc(v)
}
