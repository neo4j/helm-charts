FROM golang:1.19-alpine
RUN apk --no-cache add curl
WORKDIR reverse-proxy
COPY reverse-proxy/operations operations/
COPY reverse-proxy/proxy proxy/
COPY reverse-proxy/go.mod go.mod
COPY reverse-proxy/main.go main.go
RUN go mod download && go mod verify
RUN go build -v -o reverseproxy_linux main.go
CMD ["./reverseproxy_linux"]
