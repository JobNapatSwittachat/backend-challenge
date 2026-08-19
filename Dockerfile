# --- Build stage ---
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache dependency downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/api ./cmd/api

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /bin/api /api

EXPOSE 8080 9090
ENTRYPOINT ["/api"]
