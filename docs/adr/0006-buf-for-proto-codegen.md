# 6. Generate the gRPC contract with buf and BSR plugins

Status: Accepted

## Context

The gRPC bonus needs a `.proto` file and generated Go code. The traditional path is a local `protoc` plus `protoc-gen-go` and `protoc-gen-go-grpc` installed with `go install`. That makes the generated output depend on whatever versions each developer happens to have, and `protoc` itself is an extra native dependency to install and pin.

## Decision

Use [buf](https://buf.build). `buf.yaml` declares the module and enables the `STANDARD` lint and `FILE` breaking-change rulesets; `buf.gen.yaml` pulls the codegen plugins from the Buf Schema Registry at pinned versions (`buf.build/protocolbuffers/go:v1.36.12`, `buf.build/grpc/go:v1.6.2`) and uses managed mode to set the Go import path.

The proto lives at `proto/user/v1/user.proto`, matching its `user.v1` package as the linter requires. Generated code is committed so the repository builds without running codegen.

## Consequences

`buf` on `PATH` is the only prerequisite — no `protoc`, no plugin installs — and every machine generates byte-identical output. Managed mode keeps `option go_package` out of the `.proto`, so the file describes the contract and nothing about Go.

Three checks come along with it, wired as make targets: `proto-lint`, `proto-breaking` (compares against `main`), and `proto-check` (fails if the committed generated code is stale). The breaking check was verified by deliberately renumbering a field and confirming it fails.

Costs: generation now needs network access to the BSR, and the pinned plugin versions are one more thing to bump. Committing generated code means diffs include machine output, which is the standard trade for a repository that builds from a fresh clone.
