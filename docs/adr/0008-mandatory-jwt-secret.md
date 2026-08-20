# 8. Fail startup when `JWT_SECRET` is unset

Status: Accepted

## Context

`config.Load` refuses to start without `JWT_SECRET`, which is the right check at the right layer. But `docker-compose.yml` originally read:

```yaml
JWT_SECRET: ${JWT_SECRET:-change-me-in-production}
```

That default satisfied the check with a string published in this repository. Anyone running `docker compose up` without exporting the variable — the obvious thing to do — got a service whose tokens could be forged by anyone who had read the repo, with no warning that anything was wrong.

A guard that something upstream can quietly satisfy is not a guard.

## Decision

Make the variable mandatory in compose too, using the error form of interpolation:

```yaml
JWT_SECRET: ${JWT_SECRET:?required — put it in .env or the environment, e.g. openssl rand -hex 32}
```

Ship a `.env.example` documenting how to generate one, and point the README at it. `.env` is gitignored.

## Consequences

There is no configuration in which the service starts with a known secret. Compose fails before creating containers, with a message naming the variable and how to produce a value — verified by running `docker compose config` with the variable unset and confirming a non-zero exit.

Recommending a `.env` file rather than an inline environment variable also fixes a smaller problem: the previously documented `JWT_SECRET=$(openssl rand -hex 32) docker compose up` minted a fresh secret on every invocation, so every restart silently invalidated all outstanding tokens.

The cost is one more step before first run, and an empty `JWT_SECRET=` in `.env` fails the same way an unset one does — deliberately, since an empty HMAC key is worse than none.
