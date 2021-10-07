package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

type dictData struct {
	Name  string
	Bytes []byte
}

func generatePayloadDictionary() error {
	dict, err := ioutil.ReadFile(updatesDictionaryPath)
	if err != nil {
		return fmt.Errorf("could not read raw dictionary: %w", err)
	}

	filename := updatesDictionaryPath + ".go"
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not open dictionary file: %w", err)
	}

	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}
	t := template.Must(template.New("").Funcs(funcMap).Parse(dictionaryTemplate))
	err = t.Execute(file, dictData{Name: "Payloads", Bytes: dict})
	if err != nil {
		return fmt.Errorf("could not execute dictionary template: %w", err)
	}
}
