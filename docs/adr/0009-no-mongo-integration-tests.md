# 9. No Mongo integration tests; cover the adapter end to end

Status: Accepted

## Context

The challenge asks for unit tests with MongoDB interactions mocked where appropriate. Every layer above the driver is covered that way. That leaves the Mongo adapter itself — the code that translates driver errors into domain errors and BSON documents into entities — with no unit tests, because mocking the driver would only assert that the mock behaves like the mock.

The options were testcontainers (a real MongoDB per test run), the driver's `mtest` package (a mocked wire protocol), or covering the adapter through the running stack.

## Decision

Do not unit-test the Mongo adapter. Cover it end to end against a real MongoDB started by `docker compose`, and record the results in the [proof of work](../proof-of-work.md).

Keep the adapter thin enough that this is sufficient: it holds no business logic, only document mapping and error translation, so the behaviour worth testing is exactly the behaviour visible at the API boundary.

## Consequences

The behaviours that could actually regress — a duplicate email surfacing as 409, a malformed ObjectID as 404, an update returning the modified document — are verified against the real driver and a real server, which is stronger evidence than a mocked test would give. No test-only abstraction is added to the production code to make it mockable.

The costs are honest ones: package-level coverage for `repository/mongo` is zero, the end-to-end checks need Docker, and they are run manually rather than in CI. A driver upgrade that changed error shapes would be caught by the end-to-end run, but only when someone runs it.

If this were going to production, testcontainers in CI would be the first thing to add — it keeps the same real-driver fidelity without depending on a human remembering to run the suite.
