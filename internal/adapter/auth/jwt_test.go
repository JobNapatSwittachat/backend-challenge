package auth

import (
	"errors"
	"testing"
	"time"

	"user-management-api/internal/core/domain"
)

func newTestJWT(ttl time.Duration) *JWTService {
	return NewJWTService("test-secret", "test-issuer", ttl)
}

func TestGenerateAndVerifyRoundTrip(t *testing.T) {
	svc := newTestJWT(time.Hour)

	token, err := svc.Generate("user-123")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	subject, err := svc.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if subject != "user-123" {
		t.Errorf("want subject user-123, got %q", subject)
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	svc := newTestJWT(time.Hour)
	// Freeze time in the past for generation, so by "real" verification time
	// the token has expired.
	svc.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	token, err := svc.Generate("user-123")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	svc.now = time.Now
	if _, err := svc.Verify(token); !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("want ErrInvalidToken for expired token, got %v", err)
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	token, err := newTestJWT(time.Hour).Generate("user-123")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	other := NewJWTService("different-secret", "test-issuer", time.Hour)
	if _, err := other.Verify(token); !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("want ErrInvalidToken for wrong secret, got %v", err)
	}
}

func TestVerifyWrongIssuer(t *testing.T) {
	token, err := NewJWTService("test-secret", "someone-else", time.Hour).Generate("user-123")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := newTestJWT(time.Hour).Verify(token); !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("want ErrInvalidToken for wrong issuer, got %v", err)
	}
}

func TestVerifyGarbageToken(t *testing.T) {
	if _, err := newTestJWT(time.Hour).Verify("not.a.jwt"); !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("want ErrInvalidToken, got %v", err)
	}
}

func TestVerifyUnsignedTokenRejected(t *testing.T) {
	// A token with alg "none" (header {"alg":"none","typ":"JWT"}, empty
	// signature) must be rejected even if its payload looks valid.
	unsigned := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ1c2VyLTEyMyJ9."
	if _, err := newTestJWT(time.Hour).Verify(unsigned); !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("want ErrInvalidToken for alg=none token, got %v", err)
	}
}
