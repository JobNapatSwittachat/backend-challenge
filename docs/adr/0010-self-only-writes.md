# 10. Restrict writes to the caller's own account

Status: Accepted

## Context

The first version protected the user routes with JWT middleware and stopped there. That authenticates but does not authorize: the authenticated user ID was placed in the request context and then never read, so any valid token could `PATCH` or `DELETE` any user by ID. `GET /users` hands out every user's ID, so the IDs needed for the attack were freely available.

This is a textbook insecure direct object reference, and it is the kind of hole that a passing test suite happily hides.

The tension is that the challenge explicitly requires "list all users" and "fetch a user by ID", so locking down reads would fail the requirements.

## Decision

Split the two concerns into two middlewares. `middleware.Auth` establishes *who* the caller is; `middleware.RequireSelf` then enforces *what* they may touch, comparing the `{id}` path value against the authenticated subject and answering 403 when they differ.

Apply it to `PATCH` and `DELETE` only. Reads stay open to any authenticated user.

Two details are deliberate: a request reaching `RequireSelf` with no user in context is refused rather than passed through, so a mis-wired route fails closed; and the 403 message is identical whether or not the target exists, so it cannot be used to probe for valid IDs.

## Consequences

The policy is visible in the route table — the reader sees which routes carry the extra wrapper — and the enforcement runs before the handler, so a rejected request never reaches the service or the database. This is verified twice: end to end, where one user's attempt to modify another returns 403 and a follow-up read confirms the record is unchanged; and by deleting the middleware and watching the test suite fail with `want 403, got 204`, i.e. a successful deletion of someone else's account.

What this gives up: there is no admin role, so no one can moderate or delete another user's account. That is the correct default — an admin capability should be added deliberately, with a role claim and its own tests, not inherited by accident from missing checks.

The rule lives in the HTTP adapter rather than the core. That is the simpler placement while the write use cases are HTTP-only, and it is the wrong placement the moment gRPC grows an update or delete RPC, at which point the check belongs in the service with the actor passed in.
