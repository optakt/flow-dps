package identifier

// Network specifies which network a particular object is associated with. The
// blockchain field is always set to `flow` and the network is always set to
// `mainnet`.
//
// We are ommitting the `SubNetwork` fieldfor now, but we could use it in the
// future to distinguish between the networks of different sporks (i.e.
// `candidate4` or `mainnet-5`).
type Network struct {
	Blockchain string `json:"blockchain"`
	Network    string `json:"network"`
}
