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
	{Name: "candidate-4", API: "candidate4.archive.optakt.io:5005", First: 1065711, Last: 2033591},
	{Name: "candidate-5", API: "candidate5.archive.optakt.io:5005", First: 2033592, Last: 3187930},
	{Name: "candidate-6", API: "candidate6.archive.optakt.io:5005", First: 3187931, Last: 4132132},
	{Name: "candidate-7", API: "candidate7.archive.optakt.io:5005", First: 4132133, Last: 4972986},
	{Name: "candidate-8", API: "candidate8.archive.optakt.io:5005", First: 4972987, Last: 6483245},
	{Name: "candidate-9", API: "candidate9.archive.optakt.io:5005", First: 6483246, Last: 7601062},
	{Name: "mainnet-1", API: "mainnet1.archive.optakt.io:5005", First: 7601063, Last: 8742958},
	{Name: "mainnet-2", API: "mainnet2.archive.optakt.io:5005", First: 8742959, Last: 9737132},
	{Name: "mainnet-3", API: "mainnet3.archive.optakt.io:5005", First: 9737133, Last: 9992019},
	{Name: "mainnet-4", API: "mainnet4.archive.optakt.io:5005", First: 9992020, Last: 12020336},
	{Name: "mainnet-5", API: "mainnet5.archive.optakt.io:5005", First: 12020337, Last: 12609236},
	{Name: "mainnet-6", API: "mainnet6.archive.optakt.io:5005", First: 12609237, Last: 13404173},
	{Name: "mainnet-7", API: "mainnet7.archive.optakt.io:5005", First: 13404174, Last: 13950741},
	{Name: "mainnet-8", API: "mainnet8.archive.optakt.io:5005", First: 13950742, Last: 14892103},
	{Name: "mainnet-9", API: "mainnet9.archive.optakt.io:5005", First: 14892104, Last: math.MaxUint64},
}
