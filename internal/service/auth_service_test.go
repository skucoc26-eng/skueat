package service

import (
	"net/url"
	"testing"

	"restaurant-api/internal/config"
)

func TestGetAuthURLCarriesStateAndCallback(t *testing.T) {
	svc := NewAuthService(&config.Config{
		AppDomain:  "https://example.test",
		RestAPIKey: "test-key",
	})
	authURL, err := url.Parse(svc.GetAuthURL("test-state"))
	if err != nil {
		t.Fatal(err)
	}
	query := authURL.Query()
	if query.Get("state") != "test-state" || query.Get("redirect_uri") != "https://example.test/auth/kakao/callback" || query.Get("client_id") != "test-key" {
		t.Fatalf("OAuth query missing state, callback, or client ID: %v", query)
	}
}
