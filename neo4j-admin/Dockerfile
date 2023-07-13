ARG IMAGE
ARG DISTRIBUTION
FROM ${IMAGE}
RUN if [ "$DISTRIBUTION" = "debian" ]; then apt-get update && apt-get install -y bash netcat-openbsd curl wget gnupg apt-transport-https apt-utils lsb-release unzip less && rm -rf /var/lib/apt/lists/* ; else microdnf update -y && microdnf install -y bash nc curl wget gnupg yum-utils redhat-lsb-core unzip less ; fi
ADD backup/backup_linux bin/backup
RUN chown neo4j:neo4j bin/backup
CMD ["/bin/bash", "-c", "bin/backup"]
