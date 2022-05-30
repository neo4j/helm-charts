FROM debian:bullseye

RUN apt-get update \
    && apt-get install --assume-yes \
        python3 \
        python3-pip \
        git \
     && cd /usr/bin \
     && pip3 install git-filter-repo \
     && rm -rf /var/lib/apt/lists/*
