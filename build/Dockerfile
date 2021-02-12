# This dockerfile provides all the dependencies necessart to run go test on this project
FROM gcr.io/google.com/cloudsdktool/cloud-sdk:alpine

ENV HELM_VERSION="v3.5.2" VERIFY_CHECKSUM="true" VERIFY_SIGNATURES="true"
COPY docker_resources/get-helm-3 /tmp/get-helm-3

RUN apk add --update --no-cache go curl ca-certificates openssl && \
    ./tmp/get-helm-3 --no-sudo --version "${HELM_VERSION}" && \
    apk del curl && \
    rm -f /var/cache/apk/* \
    rm -rf /tmp/*

RUN gcloud components install kubectl

# Configure Go
ENV GOROOT /usr/lib/go
ENV GOPATH /go
ENV PATH /go/bin:$PATH

RUN mkdir -p ${GOPATH}/src ${GOPATH}/bin
