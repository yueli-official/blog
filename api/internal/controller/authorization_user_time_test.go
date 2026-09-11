package controller

import (
	"github.com/yueli-official/foundation/go/authorization"
	"testing"
	"time"
)

func TestAuthorizationGrantViewIncludesEffectiveTime(t *testing.T) {
	timestamp := time.Date(2026, 9, 8, 1, 2, 3, 0, time.UTC)
	view := authorizationGrantView(authorization.Grant{ValidFrom: timestamp})
	if !view.ValidFrom.Equal(timestamp) {
		t.Fatalf("grant effective time lost: %v", view.ValidFrom)
	}
}
