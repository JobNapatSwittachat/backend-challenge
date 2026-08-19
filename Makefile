.PHONY: build run test vet proto docker-up docker-down

build:
	go build -o bin/api ./cmd/api

run:
	JWT_SECRET=$${JWT_SECRET:-dev-secret} go run ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

# Regenerates gRPC code. Requires buf, protoc-gen-go, protoc-gen-go-grpc:
#   go install github.com/bufbuild/buf/cmd/buf@latest
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
proto:
	buf generate

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
