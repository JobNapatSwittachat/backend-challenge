# User Management API

A RESTful (+ gRPC) user management service in Go, built for the [7-solutions backend challenge](https://github.com/7-solutions/backend-challenge).

- **Go** with only the standard library `net/http` router (Go 1.22+ method/path patterns) — no web framework
- **MongoDB** via the official driver (`go.mongodb.org/mongo-driver/v2`)
- **JWT** authentication (HS256) via `github.com/golang-jwt/jwt/v5`
- **Hexagonal architecture** (ports & adapters)
- All challenge bonuses implemented: Docker/compose, interface abstraction, validation, graceful shutdown, gRPC

The part-2 design exercise lives in [docs/lottery-search-design.md](docs/lottery-search-design.md).

## Quick start

### With Docker (recommended)

```bash
JWT_SECRET=$(openssl rand -hex 32) docker compose up --build
```

API: `http://localhost:8080` · gRPC: `localhost:9090` · MongoDB: `localhost:27017`

### Locally

Requires Go ≥ 1.25 (older toolchains auto-upgrade via `GOTOOLCHAIN`) and a running MongoDB.

```bash
docker run -d -p 27017:27017 mongo:7   # if you need a MongoDB
JWT_SECRET=dev-secret go run ./cmd/api
```

### Tests

```bash
go test ./...
```

## Configuration

All via environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `JWT_SECRET` | — (**required**) | HMAC secret for signing tokens |
| `HTTP_ADDR` | `:8080` | HTTP listen address |
| `GRPC_ADDR` | `:9090` | gRPC listen address |
| `MONGO_URI` | `mongodb://localhost:27017` | MongoDB connection string |
| `MONGO_DB` | `user_management` | Database name |
| `JWT_ISSUER` | `user-management-api` | `iss` claim, validated on verify |
| `JWT_TTL` | `24h` | Token lifetime |
| `USER_COUNT_INTERVAL` | `10s` | Interval of the background user-count logger |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown grace period |

## JWT guide: generating and using tokens

**1. Register** (public):

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","email":"alice@example.com","password":"supersecret"}'
```

**2. Login to get a token** (public):

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"supersecret"}'
```

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": { "id": "68a4...", "name": "Alice", "email": "alice@example.com", "created_at": "2026-08-19T10:00:00Z" }
}
```

**3. Call protected endpoints** with the token in the `Authorization` header:

```bash
TOKEN="<token from login>"
curl -s http://localhost:8080/api/v1/users -H "Authorization: Bearer $TOKEN"
```

Token details: HS256-signed, claims are `sub` (user ID), `iss`, `iat`, `exp` (now + `JWT_TTL`). Verification enforces the HMAC algorithm family (rejects `alg=none` downgrade), issuer, and expiry — see `internal/adapter/auth/jwt.go`. Requests with a missing/invalid/expired token get `401`.

## API reference & samples

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/healthz` | – | Health check |
| `POST` | `/api/v1/auth/register` | – | Register a user |
| `POST` | `/api/v1/auth/login` | – | Login, returns JWT |
| `POST` | `/api/v1/users` | ✅ | Create a user |
| `GET` | `/api/v1/users` | ✅ | List all users |
| `GET` | `/api/v1/users/{id}` | ✅ | Fetch a user by ID |
| `PATCH` | `/api/v1/users/{id}` | ✅ | Update name and/or email |
| `DELETE` | `/api/v1/users/{id}` | ✅ | Delete a user |

**Create / Register — 201:**

```json
{ "id": "68a45f0e2c6f4b7a9d1e3c21", "name": "Alice", "email": "alice@example.com", "created_at": "2026-08-19T10:00:00.123Z" }
```

Password hashes are never serialized in any response.

**Update — `PATCH /api/v1/users/{id}` — 200:**

```bash
curl -s -X PATCH http://localhost:8080/api/v1/users/68a45f0e2c6f4b7a9d1e3c21 \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Alice Smith"}'
```

Fields are optional; omitted fields are unchanged. At least one is required.

**Delete — 204** (no body).

**Error shape** (consistent across all endpoints):

| Case | Status | Body |
| --- | --- | --- |
| Validation failure | 400 | `{"error":"validation failed: password must be at least 8 characters"}` |
| Missing/invalid token | 401 | `{"error":"invalid or expired token"}` |
| Wrong credentials | 401 | `{"error":"invalid email or password"}` |
| Unknown user | 404 | `{"error":"user not found"}` |
| Duplicate email | 409 | `{"error":"email already exists"}` |

## gRPC

`CreateUser` and `GetUser` are also exposed over gRPC on `:9090`. The contract lives in [proto/user/v1/user.proto](proto/user/v1/user.proto) and the Go code is generated from it with [buf](https://buf.build) — see the codegen section below. `GetUser` requires an `authorization: Bearer <jwt>` metadata entry, enforced by a server interceptor; `CreateUser` is public, mirroring HTTP register.

```bash
grpcurl -plaintext -d '{"name":"Bob","email":"bob@example.com","password":"supersecret"}' \
  localhost:9090 user.v1.UserService/CreateUser

grpcurl -plaintext -H "authorization: Bearer $TOKEN" -d '{"id":"<user-id>"}' \
  localhost:9090 user.v1.UserService/GetUser
```

### Proto codegen with buf

The `.proto` file is the source of truth for the gRPC contract; the Go types and client/server stubs are generated from it, never hand-written.

```bash
make proto
```

Configuration is split across [buf.yaml](buf.yaml) (module layout, `STANDARD` lint rules, breaking-change rules) and [buf.gen.yaml](buf.gen.yaml) (codegen). Two things are worth pointing out:

- **Plugins come from the Buf Schema Registry**, not the local machine: `buf.build/protocolbuffers/go` and `buf.build/grpc/go` are pinned to exact versions, so `buf` on `PATH` is the only prerequisite — no `protoc`, no `go install` of protoc plugins, and every developer and CI run generates byte-identical output.
- **Managed mode** owns the Go import path (`go_package_prefix`), so the `.proto` file stays free of language-specific options.

Generated code lands in `internal/adapter/grpc/gen/user/v1` (package `userv1`) and is committed, so the repository builds without running codegen.

| Target | Purpose |
| --- | --- |
| `make proto` | Regenerate Go code from the proto |
| `make proto-lint` | Lint the proto against buf's `STANDARD` ruleset |
| `make proto-breaking` | Fail on wire-incompatible changes vs. `main` |
| `make proto-check` | Fail if committed generated code is stale |

## Architecture

Hexagonal (ports & adapters). Dependencies point inward; the core imports no driver or framework.

```
cmd/api/                    composition root: wiring, graceful shutdown
internal/
  core/
    domain/                 entities + sentinel errors (zero dependencies)
    port/                   interfaces: UserRepository, PasswordHasher, TokenService, UserService
    service/                use cases + validation; background user counter
  adapter/
    http/
      dto/                  request/response payload structs + JSON decoding
      handler/              REST handlers (one use case call each)
      middleware/           auth, logging, recovery — one concern per file
      response/             JSON writing + domain error → HTTP status mapping
      router/               route table and middleware composition
    grpc/                   gRPC server, auth interceptor
      gen/user/v1/          buf-generated stubs (do not edit)
    repository/mongo/       MongoDB implementation of UserRepository
    auth/                   bcrypt hasher, HS256 JWT service
  config/                   env-based configuration
  testutil/                 fakes for the core ports, shared by adapter tests
```

Each HTTP concern is its own package rather than a file in one `http` package, so the dependency direction is enforced by the compiler: `router` → `handler` → `dto`/`response`, and nothing imports back up the chain.

- **HTTP/gRPC handlers** depend only on the `UserService` *port*, and the service depends only on repository/hasher/token *ports* — so unit tests swap in hand-written fakes from `internal/testutil` (standard `testing` package only, no mock framework; see `*_test.go`).
- **Wire types never touch the domain**: handlers serialize `dto.UserResponse`, built field by field from `domain.User`, so a new domain field cannot leak to clients by accident.
- **One bearer parser for both transports**: `middleware.BearerToken` is shared by the HTTP middleware and the gRPC interceptor, so `Authorization: Bearer <jwt>` is accepted (and rejected) identically over REST and gRPC.
- **Domain errors** (`ErrUserNotFound`, `ErrEmailAlreadyExists`, …) are the contract between layers; the Mongo adapter translates driver errors into them, and each transport maps them to HTTP statuses / gRPC codes in one place.
- **Concurrency task**: `internal/core/service/user_counter.go` runs in a goroutine and logs the user count every 10s; it stops cleanly on context cancellation.
- **Graceful shutdown**: `signal.NotifyContext` on SIGINT/SIGTERM → HTTP `Shutdown` with timeout, gRPC `GracefulStop`, background goroutine join (`sync.WaitGroup`), Mongo disconnect.

## Design decisions & assumptions

- **Email uniqueness is enforced by a unique Mongo index**, not a read-then-write check, so it holds under concurrent registrations; duplicate-key errors are translated to `ErrEmailAlreadyExists` (HTTP 409).
- **Passwords**: bcrypt (default cost), minimum 8 chars. Login returns the same `401 invalid email or password` whether the email is unknown or the password is wrong, to avoid account enumeration.
- **Validation** is centralized in the service layer (applies to both HTTP and gRPC): required fields, RFC-5322 email parsing (`net/mail`), name length, password length; emails are lowercased and trimmed before storage/lookup.
- **`POST /api/v1/users` vs register**: the challenge lists "create user" as an operation and also requires registration; both exist — register is public (needed to obtain a first token), create is JWT-protected. They share the same service use case.
- **`PATCH` for updates** since the API updates a subset of fields (`name`, `email`); `null`/omitted means "unchanged".
- **Authorization model**: any authenticated user can manage users (an admin-style API). Per-user ownership rules were out of scope, but the authenticated user ID is available in the request context (`UserIDFromContext`) if needed.
- **Standard library router** over a framework: Go 1.22 `ServeMux` supports method + path-variable patterns, keeping the dependency graph minimal and idiomatic.
- **Separate persistence model** (`userDocument`) from the domain entity so BSON tags and ObjectIDs never leak into the core.
- **Structured logging** with `log/slog` (JSON in production); the logging middleware records method, path, status, and execution time as required.
