package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/yueli-official/foundation/go/httpcontract"
)

type document struct {
	Paths      map[string]map[string]operation `json:"paths"`
	Components struct {
		Schemas map[string]schema `json:"schemas"`
	} `json:"components"`
}
type operation struct {
	Responses map[string]response `json:"responses"`
}
type response struct {
	Content map[string]struct {
		Schema struct {
			Ref string `json:"$ref"`
		} `json:"schema"`
	} `json:"content"`
}
type schema struct {
	Properties map[string]json.RawMessage `json:"properties"`
}

func main() {
	input := flag.String("openapi", "../contracts/openapi/blog.json", "canonical OpenAPI document")
	output := flag.String("output", "contracts/http-result/operations.json", "generated operations manifest")
	check := flag.Bool("check", false, "verify the generated manifest")
	flag.Parse()
	raw, err := os.ReadFile(*input)
	if err != nil {
		exit(err)
	}
	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		exit(fmt.Errorf("decode OpenAPI: %w", err))
	}
	manifest := httpcontract.Operations{SchemaVersion: httpcontract.OperationsSchemaVersion, Namespace: "blog", Operations: project(doc)}
	if err := manifest.Validate(); err != nil {
		exit(err)
	}
	generated, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		exit(err)
	}
	generated = append(generated, '\n')
	if *check {
		current, err := os.ReadFile(*output)
		if err != nil {
			exit(err)
		}
		if !bytes.Equal(current, generated) {
			exit(fmt.Errorf("HTTP operations manifest drifted; run go run ./cmd/httpoperations"))
		}
		return
	}
	if err := os.WriteFile(*output, generated, 0o644); err != nil {
		exit(err)
	}
}

func project(doc document) []httpcontract.Operation {
	result := make([]httpcontract.Operation, 0)
	for path, methods := range doc.Paths {
		for method, op := range methods {
			status, res, ok := successResponse(op.Responses)
			if !ok {
				continue
			}
			ref := schemaRef(res)
			if status == 204 {
				ref = ""
			}
			result = append(result, httpcontract.Operation{
				ID: operationID(method, path), Method: strings.ToUpper(method), Path: path,
				Success: httpcontract.Success{Status: status, Kind: responseKind(status, ref, doc.Components.Schemas), SchemaRef: ref},
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func successResponse(responses map[string]response) (int, response, bool) {
	statuses := make([]int, 0)
	byStatus := make(map[int]response)
	for raw, res := range responses {
		status, err := strconv.Atoi(raw)
		if err == nil && status >= 200 && status < 300 {
			statuses = append(statuses, status)
			byStatus[status] = res
		}
	}
	if len(statuses) == 0 {
		return 0, response{}, false
	}
	sort.Ints(statuses)
	return statuses[0], byStatus[statuses[0]], true
}

func schemaRef(res response) string {
	types := make([]string, 0, len(res.Content))
	for value := range res.Content {
		types = append(types, value)
	}
	sort.Strings(types)
	for _, value := range types {
		if ref := res.Content[value].Schema.Ref; ref != "" {
			return ref
		}
	}
	return ""
}

func responseKind(status int, ref string, schemas map[string]schema) string {
	if status == 204 {
		return "empty"
	}
	properties := schemas[strings.TrimPrefix(ref, "#/components/schemas/")].Properties
	_, items := properties["items"]
	_, list := properties["list"]
	_, entries := properties["entries"]
	_, total := properties["total"]
	if total && (items || list || entries) {
		return "page"
	}
	if items || list || entries {
		return "collection"
	}
	return "resource"
}

func operationID(method, path string) string {
	parts := []string{"blog", strings.ToLower(method)}
	for _, part := range strings.Split(strings.Trim(path, "/"), "/") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			part = "by-" + strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
		}
		part = strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
				return r
			}
			return '-'
		}, part)
		parts = append(parts, part)
	}
	return strings.Join(parts, ".")
}

func exit(err error) { fmt.Fprintln(os.Stderr, "httpoperations:", err); os.Exit(1) }
