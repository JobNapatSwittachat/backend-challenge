package user_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"user-management-api/internal/adapter/http/handler/user"
	"user-management-api/internal/testutil"
)

// call invokes a handler directly. pathID, when non-empty, populates the {id}
// path value that the router would otherwise set.
func call(t *testing.T, h http.HandlerFunc, method, target, body, pathID string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if pathID != "" {
		request.SetPathValue("id", pathID)
	}
	recorder := httptest.NewRecorder()
	h(recorder, request)
	return recorder
}

func handlerFor(service *testutil.UserService) *user.Handler {
	return user.New(service)
}
