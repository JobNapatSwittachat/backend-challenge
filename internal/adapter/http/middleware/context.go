// Package middleware holds the HTTP middleware chain: authentication,
// request logging, and panic recovery. Each concern lives in its own file and
// is composed by the router.
package middleware

import "context"

type contextKey string

// userIDKey holds the authenticated user's ID in the request context.
const userIDKey contextKey = "userID"

// UserIDFromContext returns the user ID that Auth put in the request context.
// The second result is false when the request did not pass through Auth.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
