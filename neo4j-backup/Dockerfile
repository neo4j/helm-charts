FROM neo4j:5.3.0-enterprise
RUN apt-get update && apt-get install -y bash curl wget gnupg apt-transport-https apt-utils lsb-release unzip less && rm -rf /var/lib/apt/lists/*
#installing gcloud
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key --keyring /usr/share/keyrings/cloud.google.gpg  add - && apt-get update -y && apt-get install google-cloud-cli -y
#installing aws cli
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && unzip awscliv2.zip && ./aws/install && rm awscliv2.zip
#installing azure cli
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash
ADD backup/backup_linux bin/backup
CMD ["/bin/bash", "-c", "bin/backup;sleep 1"]
