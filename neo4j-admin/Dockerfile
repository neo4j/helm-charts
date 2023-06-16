ARG IMAGE
FROM ${IMAGE}
RUN apt-get update && apt-get install -y bash netcat curl wget gnupg apt-transport-https apt-utils lsb-release unzip less && rm -rf /var/lib/apt/lists/*
ADD backup/backup_linux bin/backup
RUN chown neo4j:neo4j bin/backup
CMD ["/bin/bash", "-c", "bin/backup"]
