package object

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
)

// Amount is some value of a currency. It is considered invalid to specify a
// value without a currency.
type Amount struct {
	Value    string              `json:"value"`
	Currency identifier.Currency `json:"currency"`
}
