#!/usr/bin/env bash
readonly RELEASE_NAME=dedicated-storage-class

cleanup() {
    kubectl delete storageclass neo4j-data
    helm uninstall ${RELEASE_NAME}-1 ${RELEASE_NAME}-2 ${RELEASE_NAME}-3
    kubectl delete pvc data-${RELEASE_NAME}-1-0
    kubectl delete pvc data-${RELEASE_NAME}-2-0
    kubectl delete pvc data-${RELEASE_NAME}-3-0

}

cleanup
