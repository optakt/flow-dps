package ral

// WithPathFinderVersion sets a non-default version for the encoding used to
// convert ledger keys to trie paths.
func WithPathFinderVersion(version uint8) func(*Snapshot) {
	return func(s *Snapshot) {
		s.version = version
	}
}

// AtHeight sets the height of the snapshot at which we retrieve the execution
// state register values.
func AtHeight(height uint64) func(*Snapshot) {
	return func(s *Snapshot) {
		s.height = height
	}
}
