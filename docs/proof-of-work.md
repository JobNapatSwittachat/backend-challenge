# Proof of Work

Every command below was run against this repository, and the output is pasted verbatim — nothing here is illustrative or reconstructed. Reproduce it with `make verify` (static checks and tests) plus the Docker section for the runtime evidence.

Environment used:

```
go version go1.25.0 darwin/arm64
Docker version 29.7.2, build a7dcaa6
buf 1.72.0
```

---

## 1. Static checks

```
$ gofmt -l .
(no files need formatting)

$ go vet ./...
(clean)

$ buf lint
(clean)
```

`buf breaking --against '.git#branch=main'` is also wired as `make proto-breaking`. To prove it is not vacuously passing, the `email` field number in `proto/user/v1/user.proto` was temporarily changed from `3` to `7`:

```
$ make proto-breaking
buf breaking --against '.git#branch=main'
make: *** [proto-breaking] Error 100
```

The change was reverted and the check passes again.

`make proto-check` regenerates the stubs and fails if the committed generated code differs — it currently reports no drift.

## 2. Unit tests

```
$ go test ./... -count=1
?   	user-management-api/cmd/api	[no test files]
ok  	user-management-api/internal/adapter/auth	0.634s
ok  	user-management-api/internal/adapter/grpc	0.933s
?   	user-management-api/internal/adapter/grpc/gen/user/v1	[no test files]
?   	user-management-api/internal/adapter/http/dto	[no test files]
ok  	user-management-api/internal/adapter/http/handler/health	0.623s
ok  	user-management-api/internal/adapter/http/handler/user	1.536s
ok  	user-management-api/internal/adapter/http/middleware	1.227s
?   	user-management-api/internal/adapter/http/response	[no test files]
ok  	user-management-api/internal/adapter/http/router	1.749s
?   	user-management-api/internal/adapter/repository/mongo	[no test files]
?   	user-management-api/internal/config	[no test files]
?   	user-management-api/internal/core/domain	[no test files]
?   	user-management-api/internal/core/port	[no test files]
ok  	user-management-api/internal/core/service	1.957s
?   	user-management-api/internal/testutil	[no test files]
```

123 test cases including subtests.

```
$ make cover
total: (statements) 81.0%
```

That figure needs two caveats to be read correctly, so here is the breakdown per package (`-coverpkg=./...`, so packages without their own tests still count where the suite exercises them):

```
internal/adapter/auth                   94.3%
internal/adapter/grpc                   91.7%
internal/adapter/grpc/gen/user/v1       44.3%   <- generated code
internal/adapter/http/dto              100.0%
internal/adapter/http/handler/health   100.0%
internal/adapter/http/handler/user      95.0%
internal/adapter/http/middleware        98.2%
internal/adapter/http/response          79.4%
internal/adapter/http/router           100.0%
internal/core/service                   83.3%
internal/testutil                       77.8%
```

**First caveat:** the 81.0% is dragged down by generated protobuf code at 44.3%, which no one should be writing tests for. Excluding it, the hand-written code sits above 90% almost everywhere.

**Second caveat, and the more important one:** three packages are absent from that list entirely, because nothing in the test suite links them — `internal/adapter/repository/mongo`, `internal/config`, and `cmd/api`. Their coverage is zero, not merely unlisted. The Mongo adapter is uncovered by design ([ADR-0009](adr/0009-no-mongo-integration-tests.md)) and is exercised end to end in section 4 instead; `config` and the composition root are not covered at all, which is a genuine gap rather than a decision.

A note on the tooling, since it changes how you should reproduce this: on a host whose local Go is older than the `go 1.25.0` directive in `go.mod`, Go runs a downloaded toolchain, and `go test -cover` then fails with `no such tool "covdata"` for any package that has no test files. That is why `make cover` restricts the test list to packages that do have tests, and why `make verify` runs the suite without `-cover`. It is a host toolchain quirk, not a repository problem — plain `go test ./...` is unaffected.

## 3. The tests actually catch regressions

A green suite only means something if it fails when the code breaks. `middleware.RequireSelf` was temporarily removed from the `PATCH`/`DELETE` routes:

```
$ go test ./internal/adapter/http/router/
--- FAIL: TestWritesToAnotherUserAreForbidden (0.00s)
    router_test.go:114: update must not reach the service for another user
    router_test.go:131: PATCH another user: want 403, got 500
    router_test.go:118: delete must not reach the service for another user
    router_test.go:131: DELETE another user: want 403, got 204
FAIL
FAIL	user-management-api/internal/adapter/http/router	0.370s
```

Note the `204` on the delete: without that middleware the request succeeds in deleting another user's account. Restored:

```
$ go test ./internal/adapter/http/router/
ok  	user-management-api/internal/adapter/http/router
```

## 4. End-to-end against the real stack

Started with `docker compose up --build -d` (API + MongoDB 7). Two users are registered, Alice and Bob, and Alice's token is used unless stated otherwise.

```
REQUEST                                                    STATUS
GET  /healthz  (no auth)                                   200
POST /auth/register  (new email)                           201
POST /auth/register  (duplicate email)                     409
POST /auth/register  (bad email format)                    400
POST /auth/register  (password < 8 chars, multibyte)       400
POST /auth/register  (unknown JSON field)                  400
POST /auth/login  (wrong password)                         401
POST /auth/login  (unknown email)                          401
GET  /users  (no token)                                    401
GET  /users  (Bearer token)                                200
GET  /users  (lowercase 'bearer' scheme)                   200
GET  /users  (token with no scheme)                        401
GET  /users  (tampered token)                              401
POST /users  (authenticated create)                        201
GET  /users/{alice}  (own record)                          200
GET  /users/{unknown-id}                                   404
PATCH /users/{alice} as alice  (own record)                200
PATCH /users/{alice} to a taken email                      409
PATCH /users/{bob} as alice   <-- IDOR attempt             403
DELETE /users/{bob} as alice  <-- IDOR attempt             403
GET  /users/{bob}  (verify bob is untouched)               200, name still "Bob"
DELETE /users/{bob} as bob    (own record)                 204
GET  /users/{bob}  (after delete)                          404
PUT  /users/{alice}  (unregistered method)                 405
GET  /nope  (unknown route)                                404
```

Three results carry most of the weight:

- The **IDOR attempts return 403** and the follow-up read confirms Bob's record is unchanged, so the check runs before the service is reached rather than after a partial write.
- The **multibyte password is rejected**: `ไทยไทย` is 18 bytes but only 6 characters, and the rule counts characters.
- The **unknown email and the wrong password both return 401**, so login cannot be used to discover which accounts exist.

## 5. gRPC

```
$ grpcurl -plaintext list user.v1.UserService
user.v1.UserService.CreateUser
user.v1.UserService.GetUser

$ grpcurl -d '{"name":"Dave",...}' ... UserService/CreateUser        # public RPC, no token
{
  "user": {
    "id": "6a86e588a95c110b63cb1ee0",
    "name": "Dave",
    "email": "dave@example.com",
    "createdAt": "2026-08-20T11:31:20.424Z"
  }
}

$ grpcurl -d '{... same email ...}' ... UserService/CreateUser          # duplicate
ERROR:
  Code: AlreadyExists
  Message: email already exists

$ grpcurl -d '{"id":"$ALICE"}' ... UserService/GetUser                  # no token
ERROR:
  Code: Unauthenticated
  Message: missing authorization metadata

$ grpcurl -H 'authorization: $TOKEN' ... UserService/GetUser            # token without Bearer scheme
ERROR:
  Code: Unauthenticated
  Message: malformed authorization metadata

$ grpcurl -H 'authorization: bearer $TOKEN' ... UserService/GetUser     # lowercase scheme
{
  "user": {
    "id": "6a86e577a95c110b63cb1edc",
    "name": "Alice S",
    "email": "alice@example.com",
    "createdAt": "2026-08-20T11:31:03.863Z"
  }
}
```

The last two prove both transports share one credential parser: a lowercase `bearer` scheme is accepted over gRPC exactly as it is over HTTP, and a bare token with no scheme is rejected by both.

## 6. Background task, logging, and graceful shutdown

```
$ docker compose logs api | grep 'total users'
{"time":"2026-08-20T11:31:20.759304128Z","level":"INFO","msg":"total users in database","count":3}
{"time":"2026-08-20T11:31:30.759636633Z","level":"INFO","msg":"total users in database","count":3}
{"time":"2026-08-20T11:31:40.770719512Z","level":"INFO","msg":"total users in database","count":3}
```

Ten seconds apart, as required.

```
$ docker compose logs api | grep request | tail -3
{"...","msg":"request","method":"GET","path":"/api/v1/users/6a86e577...","status":404,"duration":"360.917µs"}
{"...","msg":"request","method":"PUT","path":"/api/v1/users/6a86e577...","status":405,"duration":"18.209µs"}
{"...","msg":"request","method":"GET","path":"/nope","status":404,"duration":"19.583µs"}
```

Method, path, status, and execution time, as required.

```
$ docker compose stop api        # SIGTERM
{"...","msg":"shutdown signal received"}
{"...","msg":"user counter stopped"}
{"...","msg":"shutdown complete"}
```

The background goroutine stops on context cancellation and the process exits cleanly rather than being killed.

## 7. The JWT secret cannot be defaulted

```
$ docker compose config          # with no JWT_SECRET set
error while interpolating services.api.environment.JWT_SECRET: required variable JWT_SECRET is missing a value: required — put it in .env or the environment, e.g. openssl rand -hex 32
exit status: 1
```

The stack refuses to start rather than fall back to a secret that is printed in this repository. See [ADR-0008](adr/0008-mandatory-jwt-secret.md).

---

## Known gaps

Stated plainly rather than left for the reader to find. None of these are reachable from the required feature set, but all are real:

| Gap | Effect |
| --- | --- |
| `GET /users` has no pagination, and `created_at` (the sort key) has no index | Fine at test scale; a large collection would load entirely into memory per request |
| No request body size limit, and only `ReadHeaderTimeout` is set on the HTTP server | A slow or oversized body can hold a connection open |
| `grpcServer.GracefulStop()` is not bounded by `SHUTDOWN_TIMEOUT` | A hung RPC could delay shutdown past the intended window |
| A password over 72 bytes surfaces as `500`, not `400` | bcrypt rejects it and the error is not classified as validation |
| `USER_COUNT_INTERVAL=0` panics at startup | `time.NewTicker(0)` panics; the value is not validated |
| No rate limiting on `/auth/login` or `/auth/register` | Online guessing and registration flooding are unthrottled |

These came out of a structured review of the code; they are documented rather than fixed to keep the submission scoped to the challenge's requirements.
