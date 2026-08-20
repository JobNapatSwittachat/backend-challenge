// Package dto defines the HTTP wire contract: the request bodies the API
// accepts and the response bodies it returns, one file per use case.
//
// Keeping these types out of the handler package means the JSON shape of the
// API is defined in exactly one place, and core domain entities are never
// serialized directly — a new field on domain.User cannot leak to clients by
// accident.
package dto

import (
	"encoding/json"
	"net/http"
)

// Decode reads a JSON request body into dst, rejecting unknown fields so
// typos in client payloads surface as errors instead of being ignored.
func Decode(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

// ErrorResponse is the body returned for every non-2xx response.
type ErrorResponse struct {
	Error string `json:"error"`
}
