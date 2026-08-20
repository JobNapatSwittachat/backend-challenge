.PHONY: build run test vet proto proto-lint proto-breaking proto-check docker-up docker-down

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
