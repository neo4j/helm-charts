ARG IMAGE
FROM golang:1.23.8-bullseye as build
RUN apt-get update && apt-get install -y curl && apt-get remove -y python* && apt-get autoremove -y && apt-get clean
WORKDIR backup
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
      apt-get update && apt-get install -y bash netcat-openbsd curl wget gnupg apt-transport-https apt-utils lsb-release unzip less && \
      apt-get remove -y python* && apt-get autoremove -y && apt-get clean && rm -rf /var/lib/apt/lists/* ;  \
    else  \
      #for redhat
      microdnf update -y && microdnf install -y bash nc wget gnupg yum-utils unzip less && \
      # Safely remove python packages without affecting protected packages
      microdnf remove -y python3 python3-pip python2 python2-pip --exclude=dnf,libdnf --nobest || true && \
      microdnf clean all ;  \
    fi
COPY --from=build /go/backup/backup_linux bin/backup
ENV NEO4J_server_config_strict__validation_enabled=false
RUN chown neo4j:neo4j bin/backup
CMD ["bin/backup"]
