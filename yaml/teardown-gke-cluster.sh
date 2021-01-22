#!/bin/bash

CLUSTER_NAME="sandbox"
PROJECT_ID="jenny-288814"
ZONE="europe-west1-b"
NODE_MACHINE="n1-standard-1"

gcloud container clusters delete "${CLUSTER_NAME}" --zone "${ZONE}" --project "${PROJECT_ID}"
