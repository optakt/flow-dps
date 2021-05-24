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

package scripts

import (
	"github.com/onflow/flow-go/model/flow"
)

// TODO: refactor network parameter and token address components
// => https://github.com/optakt/flow-dps/issues/42

type Params struct {
	FungibleToken flow.Address
}

func TestNet() Params {
	return Params{
		FungibleToken: flow.HexToAddress("9a0766d93b6608b7"),
	}
}

func MainNet() Params {
	return Params{
		FungibleToken: flow.HexToAddress("f233dcee88fe0abe"),
	}
}
