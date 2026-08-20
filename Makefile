.PHONY: build run test vet verify cover proto proto-lint proto-breaking proto-check docker-up docker-down

# Everything the proof of work claims for static checks and tests, in one go.
verify:
	@test -z "$$(gofmt -l .)" || { echo "gofmt: files need formatting:"; gofmt -l .; exit 1; }
	go vet ./...
	go test ./... -count=1
	buf lint

# Coverage over the packages the suite exercises. Packages with no test files
# are left out of the test list on purpose: running `-cover` on them fails on
# a host whose local Go is older than the go.mod directive, because the
# downloaded toolchain cannot resolve the covdata tool.
cover:
	go test $$(go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...) \
		-count=1 -coverpkg=./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

build:
	go build -o bin/api ./cmd/api

run:
	JWT_SECRET=$${JWT_SECRET:-dev-secret} go run ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

# Proto contract. The codegen plugins come from the Buf Schema Registry
# (buf.build), so only buf itself is needed locally:
#   go install github.com/bufbuild/buf/cmd/buf@latest
proto:
	buf generate

proto-lint:
	buf lint

# Fails if the proto contract breaks wire compatibility with the base branch.
PROTO_AGAINST ?= .git\#branch=main
proto-breaking:
	buf breaking --against '$(PROTO_AGAINST)'

# Fails if the committed Go code is out of date with the .proto files.
proto-check: proto
	git diff --exit-code -- internal/adapter/handler/grpc/gen

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
