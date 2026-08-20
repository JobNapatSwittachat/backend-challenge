# 3. Manual dependency injection, no DI framework

Status: Accepted

## Context

The hexagon in [ADR-0001](0001-hexagonal-architecture.md) means every component takes its collaborators as interfaces. That raises the question of how the object graph gets built: by hand, or with `google/wire` (compile-time codegen) or `uber/fx` (runtime container).

The graph here is six nodes: a Mongo client, a repository, a hasher, a token service, the user service, and the two servers.

## Decision

Wire everything by hand in `cmd/api/main.go`, the composition root. No DI library.

## Consequences

The entire dependency graph is readable in about twenty lines, in execution order, with no generated files and no struct tags. Construction errors are ordinary compile errors. Startup order — connect to Mongo, ensure the index, build the service, start the servers — is explicit, which matters because it is also the shutdown order in reverse.

The cost is that adding a dependency means editing `main.go` by hand, and that `run()` is doing several things at once: config, infrastructure, background task, two servers, shutdown. It is the one function in the codebase that is deliberately not decomposed, because splitting the composition root tends to hide the very ordering it exists to make obvious.

`wire` starts paying off somewhere around dozens of providers or when several binaries share overlapping graphs. Neither applies here.
