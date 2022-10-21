package trace

const (
	// App names

	// Archive API
	GetFirst                  SpanName = "archive.getFirst"
	GetLast                   SpanName = "archive.getLast"
	GetHeightForBlock         SpanName = "archive.getHeightForBlock"
	GetCommit                 SpanName = "archive.getCommit"
	GetHeader                 SpanName = "archive.getHeader"
	GetEvents                 SpanName = "archive.getEvents"
	GetRegisterValues         SpanName = "archive.getRegisterValues"
	GetCollection             SpanName = "archive.getCollection"
	ListCollectionsForHeight  SpanName = "archive.listCollectionsForHeight"
	GetGuarantee              SpanName = "archive.getGuarantee"
	GetTransaction            SpanName = "archive.getTransaction"
	GetHeightForTransaction   SpanName = "archive.getHeightForTransaction"
	ListTransactionsForHeight SpanName = "archive.listTransactionsForHeight"
	GetResult                 SpanName = "archive.getResult"
	GetSeal                   SpanName = "archive.getSeal"
	ListSealsForHeight        SpanName = "archive.listSealsForHeight"
)
