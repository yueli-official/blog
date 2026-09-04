package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/yueli-official/foundation/go/httpcontract"
)

type document struct {
	Paths      map[string]map[string]operation `json:"paths"`
	Components struct {
		Schemas map[string]json.RawMessage `json:"schemas"`
	} `json:"components"`
}
type operation struct {
	Responses map[string]json.RawMessage `json:"responses"`
}

func main() {
	openAPIPath := flag.String("openapi", "../contracts/openapi/blog.json", "OpenAPI document")
	operationsPath := flag.String("operations", "contracts/http-result/operations.json", "HTTP operations manifest")
	flag.Parse()
	raw, err := os.ReadFile(*openAPIPath)
	if err != nil {
		exit(err)
	}
	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		exit(err)
	}
	raw, err = os.ReadFile(*operationsPath)
	if err != nil {
		exit(err)
	}
	manifest, err := httpcontract.ParseOperations(raw)
	if err != nil {
		exit(err)
	}
	declared := make(map[string]struct{}, len(manifest.Operations))
	for _, item := range manifest.Operations {
		key := item.Method + " " + item.Path
		path, ok := doc.Paths[item.Path]
		if !ok {
			exit(fmt.Errorf("%s is absent from OpenAPI", key))
		}
		op, ok := path[strings.ToLower(item.Method)]
		if !ok {
			exit(fmt.Errorf("%s is absent from OpenAPI", key))
		}
		if _, ok := op.Responses[strconv.Itoa(item.Success.Status)]; !ok {
			exit(fmt.Errorf("%s status %d is absent from OpenAPI", key, item.Success.Status))
		}
		if ref := item.Success.SchemaRef; ref != "" {
			const prefix = "#/components/schemas/"
			if !strings.HasPrefix(ref, prefix) {
				exit(fmt.Errorf("%s schemaRef is not a component", key))
			}
			if _, ok := doc.Components.Schemas[strings.TrimPrefix(ref, prefix)]; !ok {
				exit(fmt.Errorf("%s schemaRef is absent", key))
			}
		}
		declared[key] = struct{}{}
	}
	for path, methods := range doc.Paths {
		for method := range methods {
			key := strings.ToUpper(method) + " " + path
			if _, ok := declared[key]; !ok {
				exit(fmt.Errorf("OpenAPI operation %s is undeclared", key))
			}
		}
	}
	fmt.Printf("HTTP operation coverage is complete (%d operations)\n", len(declared))
}

func exit(err error) { fmt.Fprintln(os.Stderr, "httpcontractcheck:", err); os.Exit(1) }
