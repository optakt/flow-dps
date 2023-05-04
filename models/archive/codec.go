package archive

// Codec represents something that can encode and decode data, as well as compress and decompress it.
type Codec interface {
	Encode(value interface{}) ([]byte, error)
	Compress(data []byte) ([]byte, error)

	Decode(data []byte, value interface{}) error
	Decompress(compressed []byte) ([]byte, error)

	Marshal(value interface{}) ([]byte, error)
	Unmarshal(compressed []byte, value interface{}) error
}
