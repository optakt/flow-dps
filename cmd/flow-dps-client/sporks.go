// Copyright 2021 Optakt Labs OÜ
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

package main

import (
	"math"
)

type Spork struct {
	Name  string
	API   string
	First uint64
	Last  uint64
}

var DefaultSporks = []Spork{
	{Name: "candidate-4", API: "candidate4.dps.optakt.io:5005", First: 1065711, Last: 2033591},
	{Name: "candidate-5", API: "candidate5.dps.optakt.io:5005", First: 2033592, Last: 3187930},
	{Name: "candidate-6", API: "candidate6.dps.optakt.io:5005", First: 3187931, Last: 4132132},
	{Name: "candidate-7", API: "candidate7.dps.optakt.io:5005", First: 4132133, Last: 4972986},
	{Name: "candidate-8", API: "candidate8.dps.optakt.io:5005", First: 4972987, Last: 6483245},
	{Name: "candidate-9", API: "candidate9.dps.optakt.io:5005", First: 6483246, Last: 7601062},
	{Name: "mainnet-1", API: "mainnet1.dps.optakt.io:5005", First: 7601063, Last: 8742958},
	{Name: "mainnet-2", API: "mainnet2.dps.optakt.io:5005", First: 8742959, Last: 9737132},
	{Name: "mainnet-3", API: "mainnet3.dps.optakt.io:5005", First: 9737133, Last: 9992019},
	{Name: "mainnet-4", API: "mainnet4.dps.optakt.io:5005", First: 9992020, Last: 12020336},
	{Name: "mainnet-5", API: "mainnet5.dps.optakt.io:5005", First: 12020337, Last: 12609236},
	{Name: "mainnet-6", API: "mainnet6.dps.optakt.io:5005", First: 12609237, Last: 13404173},
	{Name: "mainnet-7", API: "mainnet7.dps.optakt.io:5005", First: 13404174, Last: 13950741},
	{Name: "mainnet-8", API: "mainnet8.dps.optakt.io:5005", First: 13950742, Last: 14892103},
	{Name: "mainnet-9", API: "mainnet9.dps.optakt.io:5005", First: 14892104, Last: math.MaxUint64},
}
