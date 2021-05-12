package retriever

type Retriever struct {
	contracts Contracts
	scripts   Scripts
	invoke    Invoker
	convert   Converter
}

func New(contracts Contracts, scripts Scripts, invoke Invoker, convert Converter) *Retriever {

	r := &Retriever{
		contracts: contracts,
		scripts:   scripts,
		invoke:    invoke,
		convert:   convert,
	}

	return r
}
