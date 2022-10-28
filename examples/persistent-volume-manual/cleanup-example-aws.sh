#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=volume-manual

cleanup() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    helm uninstall ${RELEASE_NAME} ${RELEASE_NAME}-disk --wait --timeout 1m
    aws ec2 delete-volume --volume-id "$(aws ec2 describe-volumes --filters Name=tag:volume,Values="${RELEASE_NAME}" --no-cli-pager --query 'Volumes[0].VolumeId' --output text)"
}

cleanup
