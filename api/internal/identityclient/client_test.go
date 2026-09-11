package identityclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClientGetsPublicUserByKey(t *testing.T) {
	const userKey = "UserA123"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/v1/users/"+userKey {
			t.Fatalf("request = %s %s", request.Method, request.URL.RequestURI())
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{"user": map[string]any{
			"userKey": userKey, "handle": "alice", "displayName": "Alice",
			"avatar": map[string]any{"mediaKey": "31Pj0mXv7cfR5fdZIUvra"},
		}})
	}))
	defer server.Close()

	user := NewHTTP(server.URL).Get(context.Background(), userKey)
	if user.UserKey != userKey || user.Handle != "alice" || user.DisplayName != "Alice" {
		t.Fatalf("user = %#v", user)
	}
	if user.Avatar == nil || user.Avatar.MediaKey != "31Pj0mXv7cfR5fdZIUvra" {
		t.Fatalf("avatar = %#v", user.Avatar)
	}
}

func TestHTTPClientGetsPublicUsersInOneBatch(t *testing.T) {
	const first = "UserA123"
	const second = "UserB234"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/users" || request.URL.Query().Get("ids") != first+","+second {
			t.Fatalf("request = %s", request.URL.RequestURI())
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{"items": []map[string]any{
			{"userKey": first, "displayName": "Alice", "avatar": map[string]any{"mediaKey": "34kNV1Rw14KiopnMv5xtu"}},
			{"userKey": second, "displayName": "Bob"},
		}})
	}))
	defer server.Close()

	users := NewHTTP(server.URL).GetMany(context.Background(), []string{first, first, "", second})
	if len(users) != 2 || users[first].DisplayName != "Alice" || users[second].DisplayName != "Bob" {
		t.Fatalf("users = %#v", users)
	}
	if avatar := users[first].Avatar; avatar == nil || avatar.MediaKey != "34kNV1Rw14KiopnMv5xtu" {
		t.Fatalf("avatar = %#v", avatar)
	}
}

func TestHTTPClientFallsBackToRequestedUserKey(t *testing.T) {
	const userKey = "UserA123"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	user := NewHTTP(server.URL).Get(context.Background(), userKey)
	if user.UserKey != userKey || user.Handle != "" || user.DisplayName != "" || user.Avatar != nil || user.Cover != nil || len(user.SocialLinks) != 0 {
		t.Fatalf("fallback = %#v", user)
	}
}
