FROM golang:1.26.10-trixie AS build
WORKDIR /src
RUN git init \
    && git remote add origin https://github.com/minio/minio.git \
    && git fetch --depth=1 origin 93fb7d62d80161480bac611c09a3fcab6f1196c0 \
    && git checkout --detach FETCH_HEAD \
    && CGO_ENABLED=0 go build -ldflags "$(MINIO_RELEASE=RELEASE go run buildscripts/gen-ldflags.go)" -o /minio .

FROM alpine:3.24
COPY --from=build /minio /usr/bin/minio
ENTRYPOINT ["minio"]
