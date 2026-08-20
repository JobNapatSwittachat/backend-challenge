package health_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"user-management-api/internal/adapter/http/handler/health"
)

func TestCheck(t *testing.T) {
	recorder := httptest.NewRecorder()

	health.Check(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("want 200, got %d", recorder.Code)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"status":"ok"}` {
		t.Errorf(`want {"status":"ok"}, got %s`, got)
	}
}
