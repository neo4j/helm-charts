FROM golang:1.19-alpine
RUN apk --no-cache add curl
WORKDIR reverse-proxy
COPY operations/ operations/
COPY proxy/ proxy/
COPY go.mod go.mod
COPY main.go main.go
RUN go mod download && go mod verify
RUN go build -v -o reverseproxy_linux main.go
CMD ["./reverseproxy_linux"]
