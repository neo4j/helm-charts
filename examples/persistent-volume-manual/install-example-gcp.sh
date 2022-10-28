#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=volume-manual
readonly GCP_ZONE="$(gcloud config get compute/zone)"
readonly GCP_PROJECT="$(gcloud config get project)"

helm_install() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    gcloud compute disks create --size 10Gi --type pd-ssd "${RELEASE_NAME}"
    helm install "${RELEASE_NAME}"-disk neo4j-persistent-volume \
        --set neo4j.name="${RELEASE_NAME}" \
        --set data.driver=pd.csi.storage.gke.io \
        --set data.storageClassName="manual" \
        --set data.reclaimPolicy="Delete" \
        --set data.createPvc=true \
        --set data.createStorageClass=false \
        --set data.volumeHandle="projects/${GCP_PROJECT}/zones/${GCP_ZONE}/disks/${RELEASE_NAME}" \
        --set data.capacity.storage=10Gi
    helm install "${RELEASE_NAME}" neo4j -f examples/persistent-volume-manual/persistent-volume-manual.yaml
}

helm_install
