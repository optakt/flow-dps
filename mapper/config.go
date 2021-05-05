package mapper

// MapperOptions contains optional parameters we can set for the mapper.
type MapperConfig struct {
	CheckpointFile string
}

// WithCheckpointFile will initialize the mapper's internal trie with the trie
// from the provided checkpoint file.
func WithCheckpointFile(file string) func(*MapperConfig) {
	return func(cfg *MapperConfig) {
		cfg.CheckpointFile = file
	}
}
