package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const dictionaryTemplate = `package zbor

// {{ .Name }}Dictionary is a byte slice that contains the result of running the Zstandard training mode
// on the {{ .Name }} of the DPS index. This allows zstandard to achieve a better compression ratio, specifically for
// small data.
// See http://facebook.github.io/zstd/#small-data
// See https://github.com/facebook/zstd/blob/master/doc/zstd_compression_format.md#dictionary-format
var {{ .Name }}Dictionary = []byte{
	{{ range .Bytes }}{{ . }}, {{ end }}
}
`

type templateData struct {
	Name  string
	Bytes []byte
}

// compile takes a raw dictionary and uses a template to compile it into a Go file.
func (g *Generator) compile(dict *dictionary) error {

	// Create dictionary file.
	filename := dict.kind.String() + ".go"
	path := filepath.Join(g.cfg.DictionaryPath, filename)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not open dictionary file: %w", err)
	}
	defer file.Close()

	// Execute template using the dictionary file as the writer.
	t := template.Must(template.New("").Parse(dictionaryTemplate))
	err = t.Execute(file, templateData{Name: dict.kind.String(), Bytes: dict.raw})
	if err != nil {
		return fmt.Errorf("could not execute dictionary template: %w", err)
	}

	return nil
}
