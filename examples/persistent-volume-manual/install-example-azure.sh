#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=volume-manual
readonly AKS_CLUSTER_NAME=${1?' Azure AKS cluster name must be 1st argument'}
readonly AZ_RESOURCE_GROUP=${2?' Azure resource group must be 1st argument'}
readonly AZ_LOCATION=${3?' Azure location must be 2nd argument'}

helm_install() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    local -r node_resource_group=$(az aks show --resource-group "${AZ_RESOURCE_GROUP}" --name "${AKS_CLUSTER_NAME}" --query nodeResourceGroup -o tsv)
    local -r disk_id=$(az disk create --name "${RELEASE_NAME}" --size-gb "10" --max-shares 1 --resource-group "${node_resource_group}" --location ${AZ_LOCATION} --output tsv --query id)
    helm install "${RELEASE_NAME}"-disk neo4j-persistent-volume \
        --set neo4j.name="${RELEASE_NAME}" \
        --set data.driver=disk.csi.azure.com \
        --set data.storageClassName="manual" \
        --set data.reclaimPolicy="Delete" \
        --set data.createPvc=true \
        --set data.createStorageClass=false \
        --set data.volumeHandle="${disk_id}" \
        --set data.capacity.storage=10Gi
        helm install "${RELEASE_NAME}" neo4j -f examples/persistent-volume-manual/persistent-volume-manual.yaml
}

helm_install
