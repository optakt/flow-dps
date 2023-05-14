package storage

const (
	PrefixFirst = 1
	PrefixLast  = 2

	PrefixHeightForBlock       = 7
	PrefixHeightForTransaction = 16

	PrefixCommit  = 4
	PrefixHeader  = 3
	PrefixEvents  = 5
	PrefixPayload = 6

	PrefixTransaction = 8
	PrefixCollection  = 10
	PrefixGuarantee   = 17

	PrefixTransactionsForHeight     = 9
	PrefixTransactionsForCollection = 12
	PrefixCollectionsForHeight      = 11
	PrefixResults                   = 13

	PrefixSeal           = 14
	PrefixSealsForHeight = 15
)
