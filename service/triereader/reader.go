package triereader

// WALReader represents something that can read write-ahead log records.
type WALReader interface {
	Next() bool
	Err() error
	Record() []byte
}
