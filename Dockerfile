FROM google/cloud-sdk

RUN apt -qq update && apt upgrade -y && apt install -qq -y curl apt-transport-https curl gnupg
RUN echo "deb https://baltocdn.com/helm/stable/debian/ all main" | tee /etc/apt/sources.list.d/helm-stable-debian.list
RUN curl https://baltocdn.com/helm/signing.asc | apt-key add -
RUN apt-get update && apt-get install -qq -y helm
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

RUN mkdir /neo4j-helm
WORKDIR /neo4j-helm
ADD . /neo4j-helm

ENTRYPOINT ["/neo4j-helm/entrypoint.sh"]