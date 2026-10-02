FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM builder AS mock-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /bin/mock ./cmd/mock

FROM builder AS runner-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /bin/runner ./cmd/runner

FROM alpine:3.19 AS mock
COPY --from=mock-builder /bin/mock /bin/mock
EXPOSE 9000
CMD ["/bin/mock"]

FROM alpine:3.19 AS runner
COPY --from=runner-builder /bin/runner /bin/runner
COPY scenarios /scenarios
COPY config.json /config.json
WORKDIR /
CMD ["/bin/runner", "-config", "/config.json"]
