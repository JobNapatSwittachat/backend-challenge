# 4. Enforce email uniqueness with a MongoDB unique index

Status: Accepted

## Context

The user model requires a unique email. The obvious implementation is to look up the email and insert if nothing is found. That check-then-act is a race: two concurrent registrations with the same email can both read "not found" and both insert, and MongoDB has no cross-document transaction guarding them by default.

## Decision

Create a unique index on `email` when the repository is constructed, and let the database reject duplicates. The insert path translates the driver's duplicate-key error into `domain.ErrEmailAlreadyExists`, which the HTTP adapter maps to 409 and the gRPC adapter to `AlreadyExists`.

Emails are lowercased and trimmed in the service layer before they reach the repository, so uniqueness is effectively case-insensitive.

## Consequences

Correct under concurrency without a transaction, and the invariant survives writes that bypass this service entirely. `FindOneAndUpdate` on the update path gets the same protection for free.

The trade-offs are real but small: the check now depends on driver-specific error detection (`mongo.IsDuplicateKeyError`), which is a place a driver upgrade could silently break — the end-to-end run in the proof of work is what catches that, since the adapter has no unit tests ([ADR-0009](0009-no-mongo-integration-tests.md)). Index creation also runs at startup, adding a round trip and a failure mode if Mongo is reachable but slow.

Normalizing case in the application rather than with a collation means a document written by another client with mixed case would not collide. Given this service owns the collection, that is acceptable.
