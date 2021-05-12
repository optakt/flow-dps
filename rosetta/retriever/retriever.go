package retriever

type Retriever struct {
	contracts Contracts
	scripts   Scripts
	invoke    Invoker
	convert   Converter
}

func New(invoke Invoker, convert Converter) *Retriever {

	r := &Retriever{
		invoke:  invoke,
		convert: convert,
	}

	return r
}
