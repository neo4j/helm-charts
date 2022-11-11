#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=gds-bloom-with-license
readonly GDS_LICENSE_FILE=${1?' GDS license file path must be 1st argument'}
readonly BLOOM_LICENSE_FILE=${2?' Bloom license file path must be 1st argument'}

helm_install() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    kubectl create secret  generic --from-file=${GDS_LICENSE_FILE},${BLOOM_LICENSE_FILE} gds-bloom-license
    helm install "${RELEASE_NAME}" neo4j -f examples/bloom-gds-license/gds-bloom-with-license.yaml
}

helm_install
