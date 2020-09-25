#!/bin/bash

CLUSTER_NAME="sandbox"
PROJECT_ID="jenny-288814"
ZONE="europe-west1-b"
NODE_MACHINE="n1-standard-1"


gcloud container --project "${PROJECT_ID}" clusters create "${CLUSTER_NAME}" \
    --zone "${ZONE}" \
     --machine-type "${NODE_MACHINE}" \
     --preemptible \
     --num-nodes "3"

gcloud container clusters get-credentials "${CLUSTER_NAME}" --zone "${ZONE}" --project "${PROJECT_ID}"