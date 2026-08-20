// Package health serves the liveness endpoint. It is deliberately separate
// from the user module: it has no dependencies and must keep answering even
// if the user use cases are failing.
package health

import (
	"net/http"

	"user-management-api/internal/adapter/http/response"
)

// Check reports that the process is up and serving.
func Check(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
