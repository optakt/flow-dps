package identifier

// Operation uniquely identifies an operation within a transaction. We don't use
// a network index because we don't have a sharded chain.
type Operation struct {
	Index uint `json:"index"`
}
