FROM golang:1.22-alpine
RUN apk --no-cache add curl \
    && addgroup --gid 7474 --system neo4j \
    && adduser --uid 7474 --system --no-create-home --home "/go" --ingroup neo4j neo4j
WORKDIR reverse-proxy
COPY reverse-proxy/operations operations/
COPY reverse-proxy/proxy proxy/
COPY reverse-proxy/go.mod go.mod
COPY reverse-proxy/main.go main.go
RUN go mod download && go mod verify \
    && go build -v -o reverseproxy_linux main.go \
    && chown -R neo4j:neo4j /go && chmod -R 777 /go
CMD ["./reverseproxy_linux"]
