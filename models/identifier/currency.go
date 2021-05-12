package identifier

// Currency is composed of a canonical symbol and decimals. This decimals value
// is used to convert an amount value from atomic units (such as satoshis) to
// standard units (such as bitcoins). As monetary values in Flow are provided as
// an unsigned fixed point value with 8 decimals, we simply use the full integer
// with 8 decimals in the currency struct. The symbol is alwos `FLOW`.
//
// An example of metadata given in the Rosetta API documentation is `Issuer`.
type Currency struct {
	Symbol   string `json:"symbol"`
	Decimals uint   `json:"decimals"`
}
