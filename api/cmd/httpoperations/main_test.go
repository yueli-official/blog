package main

import (
	"encoding/json"
	"os"
	"slices"
	"sort"
	"testing"
)

func TestPersonalTokenOperationsPreserveAuthorizationErrors(t *testing.T) {
	for _, route := range []string{"GET /api/v1/internal/personal-token/permissions", "POST /api/v1/personal-token/media-authorization"} {
		for _, code := range []string{"blog.authorization_unavailable", "blog.forbidden"} {
			if !slices.Contains(operationErrors[route], code) {
				t.Errorf("%s is missing %s", route, code)
			}
		}
	}
}

func TestOperationErrorsCoverCatalogAndHaveNoStaleRoutes(t *testing.T) {
	openAPIData, err := os.ReadFile("../../../contracts/openapi/blog.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc document
	if err := json.Unmarshal(openAPIData, &doc); err != nil {
		t.Fatal(err)
	}
	operations := project(doc)
	knownRoutes := make(map[string]struct{}, len(operations))
	usedCodes := make(map[string]struct{})
	for _, operation := range operations {
		knownRoutes[operation.Method+" "+operation.Path] = struct{}{}
		for _, code := range operation.Errors {
			usedCodes[code] = struct{}{}
		}
	}
	for route := range operationErrors {
		if _, ok := knownRoutes[route]; !ok {
			t.Errorf("stale operation error declaration: %s", route)
		}
	}
	catalogData, err := os.ReadFile("../../contracts/http-result/error-catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(catalogData, &catalog); err != nil {
		t.Fatal(err)
	}
	missing := make([]string, 0)
	for _, definition := range catalog.Errors {
		if _, ok := usedCodes[definition.Code]; !ok {
			missing = append(missing, definition.Code)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("catalog errors unused by operations: %v", missing)
	}
}
