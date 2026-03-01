FROM golang:1.25.4-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/grpc_server ./cmd/grpc_server && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.20

RUN apk add --no-cache ca-certificates
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=build /out/grpc_server /app/grpc_server
COPY --from=build /out/migrate /app/migrate
COPY --from=build /src/internal/infra/persistence/postgres/migrations /app/internal/infra/persistence/postgres/migrations

USER app
ENTRYPOINT ["/app/grpc_server"]
