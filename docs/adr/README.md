# Architecture Decision Records

Each record captures one decision: the forces at the time, what was chosen, what was rejected, and what the choice costs. They are written so a reviewer can disagree with a decision without having to reverse-engineer it from the code.

Format is [Michael Nygard's](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions): Context → Decision → Consequences. Records are immutable; a later record supersedes an earlier one rather than editing it.

| # | Decision | Status |
| --- | --- | --- |
| [0001](0001-hexagonal-architecture.md) | Hexagonal architecture with ports and adapters | Accepted |
| [0002](0002-stdlib-http-router.md) | Standard library `ServeMux` instead of a web framework | Accepted |
| [0003](0003-manual-dependency-injection.md) | Manual dependency injection, no DI framework | Accepted |
| [0004](0004-unique-index-for-email.md) | Enforce email uniqueness with a MongoDB unique index | Accepted |
| [0005](0005-jwt-hs256.md) | HS256 JWTs with pinned algorithm, issuer, and expiry | Accepted |
| [0006](0006-buf-for-proto-codegen.md) | Generate the gRPC contract with buf and BSR plugins | Accepted |
| [0007](0007-package-per-module-file-per-use-case.md) | One package per module, one file per use case | Accepted |
| [0008](0008-mandatory-jwt-secret.md) | Fail startup when `JWT_SECRET` is unset | Accepted |
| [0009](0009-no-mongo-integration-tests.md) | No Mongo integration tests; cover the adapter end to end | Accepted |
| [0010](0010-self-only-writes.md) | Restrict writes to the caller's own account | Accepted |
