package rest

type RegisterResponse struct {
	Height uint64 `json:"height"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}
