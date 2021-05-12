package identifier

// Block uniquely identifies a block in a particular network. As the view is not
// unique between sporks, index refers to the block height.
type Block struct {
	Index uint64 `json:"index"`
	Hash  string `json:"hash"`
}
