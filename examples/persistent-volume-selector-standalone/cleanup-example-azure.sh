#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=volume-selector

cleanup() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    helm uninstall ${RELEASE_NAME} ${RELEASE_NAME}-disk
    kubectl delete pvc data-${RELEASE_NAME}-0
    az disk delete --name ${RELEASE_NAME} -y
}

cleanup
