# 7. One package per module, one file per use case

Status: Accepted

## Context

The first version of the HTTP adapter was a single `http` package holding `router.go`, `middleware.go`, `response.go`, and `user_handler.go`. That package also had to import `net/http`, shadowing its own name, and nothing stopped a handler from reaching into routing or vice versa.

## Decision

Two rules, applied consistently:

1. **A package per module or concern.** `dto`, `response`, `middleware`, `router`, `handler/user`, `handler/health`. Adding a second entity means adding `handler/<entity>/`, not growing an existing file.
2. **A file per use case.** `register.go`, `login.go`, `update.go`, `delete.go`, each with its test file beside it, lined up one-to-one across the service, HTTP, and gRPC layers.

File names are lowercase with no separators, and are not prefixed with the module — `user.Handler.Register` lives in `handler/user/register.go`, since the package already supplies that context.

## Consequences

The dependency direction is enforced by the compiler rather than by discipline: `router` → `handler` → `dto`/`response`, and nothing imports back up. Changing one use case touches one small file per layer, so diffs are legible and merge conflicts are rare.

Two deliberate exceptions, both about not fragmenting for its own sake: `query.go` groups the three read-only use cases, which are one-line pass-throughs with no rules of their own, and the Mongo repository stays in one file because it implements a single port as one cohesive unit of driver-error translation.

The cost is more files and longer import paths, and a reader who wants the whole HTTP story must open several directories. A known wart left in place: `adapter/grpc` imports `adapter/http/middleware` to share the bearer-token parser, so one driving adapter depends on another. The right home is a small shared package on the driving side; it is one file move, not done here to keep the change surface small.
