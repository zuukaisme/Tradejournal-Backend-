package auth

import (
	"strings"
	"testing"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	handler := NewHandler(nil, "test-secret-with-more-than-thirty-two-characters")
	token, expiresIn, err := handler.accessToken("user-123", "account-456")
	if err != nil {
		t.Fatalf("create access token: %v", err)
	}
	if expiresIn != 900 {
		t.Fatalf("expected 900 seconds, got %d", expiresIn)
	}
	parsed, err := handler.parseAccessToken(token)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if parsed.UserID != "user-123" || parsed.AccountID != "account-456" {
		t.Fatalf("unexpected claims: %+v", parsed)
	}
}

func TestAccessTokenRejectsTampering(t *testing.T) {
	handler := NewHandler(nil, "test-secret-with-more-than-thirty-two-characters")
	token, _, err := handler.accessToken("user-123", "account-456")
	if err != nil {
		t.Fatalf("create access token: %v", err)
	}
	parts := strings.Split(token, ".")
	parts[1] = parts[1] + "x"
	if _, err := handler.parseAccessToken(strings.Join(parts, ".")); err == nil {
		t.Fatal("expected modified token to be rejected")
	}
}

func TestAccessTokenRejectsMalformedInput(t *testing.T) {
	handler := NewHandler(nil, "test-secret-with-more-than-thirty-two-characters")
	for _, token := range []string{"", "not-a-token", "a.b.c", "a.b.c.d"} {
		if _, err := handler.parseAccessToken(token); err == nil {
			t.Fatalf("expected token %q to be rejected", token)
		}
	}
}
