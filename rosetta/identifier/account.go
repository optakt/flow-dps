package identifier

// Account uniquely identifies an account within a network. All fields in the
// account identifier are utilized to determine uniqueness, including the
// metadata field, if populated. We don't use sub-accounts in this
// implementation for now, though we will probably have to add it to support
// staking on Coinbase in the future.
type Account struct {
	Address string `json:"address"`
}
