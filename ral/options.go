package ral

// WithPathFinderVersion sets a non-default version for the encoding used to
// convert ledger keys to trie paths.
func WithPathFinderVersion(version uint8) func(*Ledger) {
	return func(s *Ledger) {
		s.version = version
	}
}
