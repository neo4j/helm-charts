#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=volume-selector

cleanup() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    for i in {1..3}; do
        helm uninstall ${RELEASE_NAME}-${i} ${RELEASE_NAME}-disk-${i}
        kubectl delete pvc data-${RELEASE_NAME}-${i}-0
        aws ec2 delete-volume --volume-id "$(aws ec2 describe-volumes --filters Name=tag:volume,Values="${RELEASE_NAME}-${i}" --no-cli-pager --query 'Volumes[0].VolumeId' --output text)"
    done
}

cleanup
