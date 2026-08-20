# 1. Hexagonal architecture with ports and adapters

Status: Accepted

## Context

The challenge asks for a user API over MongoDB with JWT auth, and lists hexagonal architecture as a bonus: separate domain, application, and infrastructure layers, with business logic decoupled from frameworks and drivers.

The forces that actually matter here are testability and the fact that the same use cases must be reachable from two transports (REST and gRPC) and one background task. A design where the handler talks to the driver directly would force every test to stand up a database, and would duplicate the rules across transports.

## Decision

Structure the code as a hexagon:

- `internal/core/domain` holds entities and sentinel errors and imports nothing.
- `internal/core/port` declares the interfaces. `UserService` is the **driving** port (what the outside world calls); `UserRepository`, `PasswordHasher`, and `TokenService` are **driven** ports (what the core calls out to).
- `internal/core/service` implements the driving port using only driven ports.
- `internal/adapter/**` holds the implementations: HTTP and gRPC on the driving side, MongoDB, bcrypt, and JWT on the driven side.
- `cmd/api` is the only place that knows which concrete adapter is which.

Dependencies point inward. The compiler enforces it: nothing under `core/` imports anything under `adapter/`.

## Consequences

Validation, normalization, and the "don't reveal whether an email exists" rule live in one place and apply to both transports automatically. Tests for the core need no database, no HTTP server, and no clock. Swapping MongoDB for Postgres would touch one package.

The cost is indirection: reading one request end to end means opening a handler, a port, and a service. For an application this size that is a real tax, paid deliberately because the challenge asks for the structure and because two transports genuinely share the use cases.

One thing this does not buy: the ports are still shaped around what this application needs today. A second consumer with different needs would likely change the port rather than reuse it as-is.
