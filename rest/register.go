package rest

type RegisterResponse struct {
	Height uint64
	Key    []byte
	Value  []byte
}
