// Copyright 2021 Alvalor S.A.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package converter

import (
	"strconv"

	"github.com/awfm9/flow-dps/models/identifier"
	"github.com/awfm9/flow-dps/models/rosetta"
)

type Converter struct {
}

func New() *Converter {

	c := &Converter{}

	return c
}

func (c *Converter) Balance(currency identifier.Currency, balance uint64) rosetta.Amount {
	amount := rosetta.Amount{
		Value:    strconv.FormatUint(balance, 10),
		Currency: currency,
	}
	return amount
}
