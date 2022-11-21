# This dockerfile provides all the dependencies necessart to run go test on this project
FROM gcr.io/google.com/cloudsdktool/cloud-sdk:slim

ENV HELM_VERSION="v3.9.4" VERIFY_CHECKSUM="true" VERIFY_SIGNATURES="false" GO_VERSION="1.19" SHELL=bash
COPY docker_resources/* /tmp/

# use bash as the default shell (https://unix.stackexchange.com/questions/442510/how-to-use-bash-for-sh-in-ubuntu)
RUN cd /bin && ln -s bash /bin/sh.bash && mv /bin/sh.bash /bin/sh

# Install kubectl, helm go and dev/install dependencies
RUN apt-get update && apt-get install -y \
  bash \
  curl \
  ca-certificates \
  openssl \
  google-cloud-sdk-gke-gcloud-auth-plugin \
  kubectl \
  awscli \
  jq \
  && ./tmp/get-helm-3 --no-sudo --version "${HELM_VERSION}" \
  && ./tmp/go_linux_installer -v -version "${GO_VERSION}" \
  && apt-get -y --purge remove curl \
  && rm -rf /var/lib/apt/lists/* \
  && rm -rf /tmp/*

ENV USE_GKE_GCLOUD_AUTH_PLUGIN True
# Configure Go
ENV GOPATH /root/go
ENV PATH /root/go/bin:/root/.go/bin:$PATH
RUN mkdir -p ${GOPATH}/src ${GOPATH}/bin
