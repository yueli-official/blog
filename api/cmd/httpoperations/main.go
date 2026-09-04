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
			key := strings.ToUpper(method) + " " + path
			result = append(result, httpcontract.Operation{
				ID: operationID(method, path), Method: strings.ToUpper(method), Path: path,
				Success: httpcontract.Success{Status: status, Kind: responseKind(status, ref, doc.Components.Schemas), SchemaRef: ref},
				Errors:  operationErrors[key],
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

var operationErrors = map[string][]string{
	"GET /api/v1/authors/{id}":                                                       {"blog.not_found"},
	"GET /api/v1/posts/{slug}":                                                       {"blog.not_found"},
	"GET /api/v1/posts/{slug}/comments":                                              {"blog.not_found", "blog.invalid_input"},
	"GET /api/v1/posts/{slug}/related":                                               {"blog.not_found"},
	"GET /api/v1/posts/{slug}/siblings":                                              {"blog.not_found"},
	"GET /api/v1/series/{slug}":                                                      {"blog.not_found"},
	"GET /api/v1/subscribe/confirm":                                                  {"blog.invalid_input"},
	"GET /api/v1/dashboard/overview":                                                 {"blog.forbidden", "blog.authorization_unavailable", "blog.invalid_input"},
	"GET /api/v1/me/profile":                                                         {"blog.forbidden"},
	"GET /api/v1/posts/mine":                                                         {"blog.forbidden", "blog.authorization_unavailable", "blog.invalid_input"},
	"GET /api/v1/posts/{id}/revisions":                                               {"blog.forbidden", "blog.authorization_unavailable", "blog.not_found"},
	"GET /api/v1/comments/mine":                                                      {"blog.forbidden", "blog.authorization_unavailable", "blog.invalid_input"},
	"POST /api/v1/authorization/setup/claim":                                         {"blog.authorization_unavailable", "blog.initial_administrator_already_claimed"},
	"GET /api/v1/authorization/setup":                                                {"blog.authorization_unavailable"},
	"GET /api/v1/authorization/applications/mine":                                    {"blog.authorization_unavailable", "blog.forbidden"},
	"GET /api/v1/authorization/manage/console":                                       {"blog.authorization_unavailable", "blog.forbidden"},
	"GET /api/v1/authorization/requestable-roles":                                    {"blog.authorization_unavailable", "blog.forbidden"},
	"POST /api/v1/authorization/applications":                                        {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"POST /api/v1/authorization/applications/{id}/withdraw":                          {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.not_found"},
	"POST /api/v1/authorization/manage/applications/{id}/review":                     {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.not_found"},
	"POST /api/v1/authorization/manage/grants":                                       {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"DELETE /api/v1/authorization/manage/grants/{id}":                                {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.not_found"},
	"POST /api/v1/authorization/manage/policies/drafts":                              {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"POST /api/v1/authorization/manage/policies/{revision}/activate":                 {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"POST /api/v1/authorization/manage/policies/{revision}/preview":                  {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"POST /api/v1/authorization/manage/policies/{revision}/roles":                    {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"POST /api/v1/authorization/manage/policies/{revision}/roles/{role}/retire":      {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"POST /api/v1/authorization/manage/policies/{revision}/validate":                 {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"PUT /api/v1/authorization/manage/policies/{revision}/automatic/{rule}":          {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"PUT /api/v1/authorization/manage/policies/{revision}/roles/{role}/capabilities": {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"POST /api/v1/posts":                                                             {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.slug_taken"},
	"PATCH /api/v1/posts/{id}":                                                       {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.invalid_state", "blog.not_found", "blog.slug_taken"},
	"DELETE /api/v1/posts/{id}":                                                      {"blog.authorization_unavailable", "blog.forbidden", "blog.not_found"},
	"DELETE /api/v1/posts/{id}/permanent":                                            {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_state", "blog.not_found"},
	"POST /api/v1/posts/{id}/restore":                                                {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_state", "blog.not_found"},
	"POST /api/v1/posts/batch":                                                       {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"PUT /api/v1/posts/{id}/flags":                                                   {"blog.authorization_unavailable", "blog.forbidden", "blog.not_found"},
	"PUT /api/v1/posts/{id}/seo":                                                     {"blog.authorization_unavailable", "blog.forbidden", "blog.not_found"},
	"PUT /api/v1/posts/{id}/series":                                                  {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.not_found"},
	"PUT /api/v1/posts/{id}/taxonomies":                                              {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.invalid_state", "blog.not_found"},
	"POST /api/v1/posts/{id}/revisions/{revId}/restore":                              {"blog.authorization_unavailable", "blog.forbidden", "blog.not_found"},
	"POST /api/v1/posts/{id}/cover":                                                  {"blog.authorization_unavailable", "blog.forbidden", "blog.not_found", "blog.asset_too_large", "blog.upstream_failed"},
	"POST /api/v1/posts/{id}/cover/finalize":                                         {"blog.authorization_unavailable", "blog.forbidden", "blog.not_found", "blog.asset_too_large", "blog.upstream_failed"},
	"POST /api/v1/images":                                                            {"blog.authorization_unavailable", "blog.forbidden", "blog.asset_too_large", "blog.upstream_failed"},
	"POST /api/v1/images/finalize":                                                   {"blog.authorization_unavailable", "blog.forbidden", "blog.asset_too_large", "blog.upstream_failed"},
	"POST /api/v1/posts/{slug}/comments":                                             {"blog.not_found", "blog.invalid_input", "blog.comments_closed", "blog.comment_rejected", "blog.rate_limited", "blog.challenge_required", "blog.abuse_unavailable", "blog.abuse_attempt_replayed"},
	"POST /api/v1/posts/{slug}/like":                                                 {"blog.forbidden", "blog.not_found"},
	"POST /api/v1/posts/{slug}/bookmark":                                             {"blog.forbidden", "blog.not_found"},
	"POST /api/v1/posts/{slug}/view":                                                 {"blog.invalid_input", "blog.not_found"},
	"PATCH /api/v1/comments/{id}":                                                    {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.comment_not_found"},
	"DELETE /api/v1/comments/{id}":                                                   {"blog.authorization_unavailable", "blog.forbidden", "blog.comment_not_found"},
	"POST /api/v1/series":                                                            {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.slug_taken"},
	"PATCH /api/v1/series/{id}":                                                      {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.not_found", "blog.slug_taken"},
	"DELETE /api/v1/series/{id}":                                                     {"blog.authorization_unavailable", "blog.forbidden", "blog.not_found"},
	"POST /api/v1/taxonomies":                                                        {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input"},
	"PATCH /api/v1/taxonomies/{id}":                                                  {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.invalid_state", "blog.not_found"},
	"DELETE /api/v1/taxonomies/{id}":                                                 {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_state", "blog.not_found"},
	"POST /api/v1/taxonomies/{id}/merge":                                             {"blog.authorization_unavailable", "blog.forbidden", "blog.invalid_input", "blog.invalid_state", "blog.not_found"},
	"POST /api/v1/subscribe":                                                         {"blog.invalid_input"},
	"POST /api/v1/unsubscribe":                                                       {"blog.invalid_input"},
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
