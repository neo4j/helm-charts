#!/bin/bash

# change the cluster name to whatever you want
export CLUSTER_NAME=sandbox

# There doesn't seem to be a way of configuring by cli. The config file is a bit empty for now but we may need to add stuff there later.
kind create cluster --name ${CLUSTER_NAME} --config=./kindconfig.yaml