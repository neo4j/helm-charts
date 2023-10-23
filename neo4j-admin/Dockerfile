ARG IMAGE
FROM golang:1.19-bullseye as build
RUN apt-get install -y curl
WORKDIR backup
COPY backup/aws aws/
COPY backup/azure azure/
COPY backup/gcp gcp/
COPY backup/common common/
COPY backup/main main/
COPY backup/neo4j-admin neo4j-admin/
COPY backup/go.mod go.mod
RUN go mod tidy && go mod download && go mod verify
RUN env GOOS=linux GOARCH=amd64 go build -v -o backup_linux main/*

ARG IMAGE
FROM ${IMAGE} as final
ARG DISTRIBUTION
RUN \
    if [ "${DISTRIBUTION}" = "debian" ]; then  \
      apt-get update && apt-get install -y bash netcat-openbsd curl wget gnupg apt-transport-https apt-utils lsb-release unzip less && rm -rf /var/lib/apt/lists/* ;  \
    else  \
      #for redhat
      microdnf update -y && microdnf install -y bash nc curl wget gnupg yum-utils redhat-lsb-core unzip less ;  \
    fi
COPY --from=build /go/backup/backup_linux bin/backup
RUN chown neo4j:neo4j bin/backup
CMD ["bin/backup"]
