package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// trainDictionary runs the zstd command to train a dictionary of the given kind and size,
// and returns it as a dictionary structure.
func (g *Generator) trainDictionary(kind DictionaryKind, size int) (*dictionary, error) {

	// List all samples within the sample path, to be given to the training command.
	path := filepath.Join(g.cfg.SamplePath, string(kind), "*")
	samples, err := filepath.Glob(path)
	if err != nil {
		return nil, fmt.Errorf("could not find any samples in path %s: %w", path, err)
	}

	// Build the training command.
	rawDictPath := filepath.Join(g.cfg.DictionaryPath, string(kind))
	command := []string{"zstd", "--train", "--maxdict", fmt.Sprint(size), "-o", rawDictPath}
	command = append(command, samples...)

	train := exec.Command(command[0], command[1:]...)

	// Run the training.
	err = train.Run()
	if err != nil {
		return nil, fmt.Errorf("could not train dictionary: %w", err)
	}

	// Read the resulting raw dictionary.
	raw, err := os.ReadFile(rawDictPath)
	if err != nil {
		return nil, fmt.Errorf("could not read raw dictionary: %w", err)
	}

	// Remove raw dictionary since we have its bytes in memory.
	err = os.RemoveAll(rawDictPath)
	if err != nil {
		return nil, fmt.Errorf("could not delete raw dictionary from filesystem: %w", err)
	}

	dict := dictionary{
		kind: kind,
		raw:  raw,
		size: size,
	}

	return &dict, nil
}
