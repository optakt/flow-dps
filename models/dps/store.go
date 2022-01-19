package dps

type Store interface {
	Save(key [32]byte, payload []byte)
	Retrieve(key [32]byte) ([]byte, error)
	Close() error
}
