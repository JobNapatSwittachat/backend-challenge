# 2. Standard library `ServeMux` instead of a web framework

Status: Accepted

## Context

The API needs method-based routing and a path variable (`/users/{id}`). The reflexive choice in Go is Gin, Echo, or chi. Go 1.22 added method and wildcard patterns to `net/http.ServeMux`, which covers exactly these two needs.

## Decision

Use `http.NewServeMux()` with Go 1.22+ patterns such as `mux.Handle("PATCH /api/v1/users/{id}", ...)`, and plain `func(http.Handler) http.Handler` middleware.

## Consequences

The dependency graph stays small: the only direct third-party dependencies are the MongoDB driver, `golang-jwt`, `x/crypto`, and the gRPC/protobuf runtime — all of them load-bearing. Handlers are ordinary `http.HandlerFunc`s, so they are testable with `httptest` and carry no framework types. A reviewer needs no framework knowledge to read the routing.

What is given up: no built-in binding/validation helpers (validation is in the service layer anyway, which is where it belongs for gRPC to share it), no route groups, and no automatic parameter coercion. `ServeMux` also returns 405 for a matched path with an unregistered method, which is correct but easy to overlook — it is covered by a test.

If the API grew middleware-per-group needs or dozens of routes, chi would become the better trade. At eight routes it is not.
