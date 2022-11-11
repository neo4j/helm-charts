#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=gds-no-license

helm_install() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    helm install "${RELEASE_NAME}" neo4j -f examples/bloom-gds-license/gds-no-license.yaml
}

helm_install
