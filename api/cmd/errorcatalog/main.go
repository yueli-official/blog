package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/yueli-official/foundation/go/httpcontract"
)

type legacyDocument struct {
	SchemaVersion string        `json:"schemaVersion"`
	Errors        []legacyError `json:"errors"`
}

type legacyError struct {
	Code   string `json:"code"`
	Status int    `json:"status"`
}

func main() {
	input := flag.String("input", "contracts/http-result/error-catalog.json", "Foundation error catalog source")
	output := flag.String("output", "contracts/errors/catalog.json", "legacy catalog projection")
	check := flag.Bool("check", false, "verify the committed projection")
	flag.Parse()
	data, err := os.ReadFile(*input)
	if err != nil {
		exit(err)
	}
	catalog, err := httpcontract.ParseErrorCatalog(data)
	if err != nil {
		exit(err)
	}
	legacy := legacyDocument{SchemaVersion: "blog.yueli.dev/error-catalog/v1", Errors: make([]legacyError, 0, len(catalog.Errors))}
	for _, definition := range catalog.Errors {
		legacy.Errors = append(legacy.Errors, legacyError{Code: definition.Code, Status: definition.Status})
	}
	data, err = json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		exit(err)
	}
	data = append(data, '\n')
	if *check {
		current, err := os.ReadFile(*output)
		if err != nil {
			exit(err)
		}
		if !bytes.Equal(current, data) {
			exit(fmt.Errorf("legacy error catalog drifted; run go run ./cmd/errorcatalog"))
		}
		return
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		exit(err)
	}
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, "errorcatalog:", err)
	os.Exit(1)
}
