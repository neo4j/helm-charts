#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=dedicated-storage-class

helm_install() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    kubectl apply -f examples/dedicated-storage-class-cluster/gcp-storage-class.yaml
    for i in {1..3}; do
        helm install "${RELEASE_NAME}-${i}" neo4j -fexamples/dedicated-storage-class-cluster/dedicated-storage-class.yaml
    done
}

helm_install
