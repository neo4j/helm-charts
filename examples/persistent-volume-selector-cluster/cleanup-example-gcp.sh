#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=volume-selector

cleanup() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    for i in {1..3}; do
        helm uninstall ${RELEASE_NAME}-${i} ${RELEASE_NAME}-disk-${i}
        kubectl delete pvc data-${RELEASE_NAME}-${i}-0
        gcloud compute disks delete ${RELEASE_NAME}-${i} --quiet
    done
}

cleanup
