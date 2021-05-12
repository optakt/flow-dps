package identifier

// Transaction uniquely identifies a transaction in a particular network and
// block.
type Transaction struct {
	Hash string `json:"hash"`
}
