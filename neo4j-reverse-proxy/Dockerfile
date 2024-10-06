FROM golang:1.22-alpine
RUN apk --no-cache add curl \
    && addgroup --gid 7474 --system neo4j \
    && adduser --uid 7474 --system --no-create-home --home "/go" --ingroup neo4j neo4j

WORKDIR /go/reverse-proxy

USER neo4j

COPY reverse-proxy/operations operations/
COPY reverse-proxy/proxy proxy/
COPY reverse-proxy/go.mod go.mod
COPY reverse-proxy/main.go main.go

USER root
RUN go mod download && go mod verify \
    && go build -v -o reverseproxy_linux main.go \
    && chown -R neo4j:neo4j /go/reverse-proxy && chmod -R 777 /go

USER neo4j

CMD ["./reverseproxy_linux"]