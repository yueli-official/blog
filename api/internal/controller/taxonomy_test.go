package controller

import (
	"context"
	"testing"

	foundationauth "github.com/yueli-official/foundation/go/auth"
	v1 "platform/products/blog/api/api/v1"
)

// Non-admin must be rejected before the service is touched, so a nil svc is safe.
func TestCreateTaxonomyRequiresAdmin(t *testing.T) {
	c := &Taxonomy{}
	userCtx := foundationauth.NewContext(context.Background(),
		&foundationauth.Principal{Subject: "u-plain", Roles: []string{"user"}})
	if _, err := c.CreateTaxonomy(userCtx, &v1.CreateTaxonomyReq{Name: "Go", Taxonomy: "category"}); err == nil {
		t.Fatal("non-admin CreateTaxonomy should be forbidden")
	}
}
