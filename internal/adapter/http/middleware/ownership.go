package middleware

import (
	"net/http"

	"user-management-api/internal/adapter/http/response"
)

// RequireSelf rejects a request whose {id} path value is not the
// authenticated user, so one account cannot modify or delete another.
//
// It must be applied inside Auth, which is what puts the user ID in the
// request context; a request that reaches it without one is refused rather
// than allowed through.
func RequireSelf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "missing or malformed Authorization header")
			return
		}

		if r.PathValue("id") != userID {
			// Deliberately the same message whether or not the target user
			// exists, so this cannot be used to probe for valid IDs.
			response.Error(w, http.StatusForbidden, "you may only modify your own account")
			return
		}

		next.ServeHTTP(w, r)
	})
}
