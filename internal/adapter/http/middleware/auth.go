package middleware

import (
	"net/http"
	"strings"

	"user-management-api/internal/adapter/http/response"
	"user-management-api/internal/core/port"
)

// Auth validates the "Authorization: Bearer <token>" header and injects the
// authenticated user ID into the request context.
func Auth(tokens port.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := BearerToken(r.Header.Get("Authorization"))
			if !ok {
				response.Error(w, http.StatusUnauthorized, "missing or malformed Authorization header")
				return
			}

			userID, err := tokens.Verify(token)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), userID)))
		})
	}
}

// BearerToken extracts the credential from an Authorization header value.
// The scheme is matched case-insensitively per RFC 7235, and a header without
// the Bearer scheme is rejected. It is exported so the gRPC interceptor parses
// credentials exactly the same way.
func BearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}
