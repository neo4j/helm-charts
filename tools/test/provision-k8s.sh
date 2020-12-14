#!/bin/bash
#
# This script is intended to be used for internal testing only, to create the artifacts necessary for 
# testing and deploying this code in a sample GKE cluster.
INSTANCE=${1:-helm-test}
ZONE=us-central1-a
MACHINE=n1-highmem-4
NODES=1
API=beta

echo "Creating GKE instance $INSTANCE..."
gcloud beta container clusters create $INSTANCE \
    --zone "$ZONE" \
    --project $PROJECT \
    --machine-type $MACHINE \
    --num-nodes $NODES \
    --enable-ip-alias \
    --no-enable-autoupgrade \
    --max-nodes "10" \
    --enable-autoscaling

echo "Fixing kubectl credentials to talk to $INSTANCE"
gcloud container clusters get-credentials $INSTANCE \
   --zone $ZONE \
   --project $PROJECT

# Configure local auth of docker so that we can use regular
# docker commands to push/pull from our GCR setup.
# gcloud auth configure-docker

# Bootstrap RBAC cluster-admin for your user.
# More info: https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control
echo "Creating  role binding for $INSTANCE"
kubectl create clusterrolebinding cluster-admin-binding \
  --clusterrole cluster-admin --user $(gcloud config get-value account)

echo "Done"
exit 0