# This dockerfile provides all the dependencies necessart to run go test on this project
FROM gcr.io/google.com/cloudsdktool/cloud-sdk:alpine

RUN gcloud components install kubectl
RUN apk add --no-cache go

# Configure Go
ENV GOROOT /usr/lib/go
ENV GOPATH /go
ENV PATH /go/bin:$PATH

RUN mkdir -p ${GOPATH}/src ${GOPATH}/bin
