FROM golang:1.23.8-alpine AS build
WORKDIR /go/reverse-proxy
COPY reverse-proxy/operations operations/
COPY reverse-proxy/proxy proxy/
COPY reverse-proxy/go.mod go.mod
COPY reverse-proxy/main.go main.go
RUN go mod download && go mod verify \
    && go build -v -o reverseproxy_linux main.go

FROM alpine:3.21.3 AS run
RUN addgroup --gid 7474 --system neo4j \
    && adduser --uid 7474 --system --no-create-home neo4j
WORKDIR /reverse-proxy
COPY --from=build --chown=neo4j:neo4j --chmod=777 /go/reverse-proxy/reverseproxy_linux reverseproxy_linux
USER neo4j
CMD ["./reverseproxy_linux"]
