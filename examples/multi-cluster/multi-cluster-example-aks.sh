#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly AKS_LOCATION=uksouth
readonly AKS_GROUP=multiClusterGroup
readonly VNET_NAME=multiClusterVnet
readonly GATEWAY_NAME=multiClusterGateway

setup_clusters() {
    echo "Creating resource group ${AKS_GROUP}"
    az group create --name ${AKS_GROUP} --location ${AKS_LOCATION}

    echo "Creating virtual network multiClusterVnet"
    az network vnet create \
      --name ${VNET_NAME} \
      --resource-group ${AKS_GROUP} \
      --address-prefixes 10.30.0.0/16

    echo "Creating subnet1"
    local -r subnet1_id=$(az network vnet subnet create -g ${AKS_GROUP} --vnet-name ${VNET_NAME} -n subnet1 --address-prefixes 10.30.1.0/24 --output tsv --query id)
    echo "Creating subnet2"
    local -r subnet2_id=$(az network vnet subnet create -g ${AKS_GROUP} --vnet-name ${VNET_NAME} -n subnet2 --address-prefixes 10.30.2.0/24 --output tsv --query id)
    echo "Creating subnet3"
    local -r subnet3_id=$(az network vnet subnet create -g ${AKS_GROUP} --vnet-name ${VNET_NAME} -n subnet3 --address-prefixes 10.30.3.0/24 --output tsv --query id)
    echo "Creating subnet4"
    az network vnet subnet create -g ${AKS_GROUP} --vnet-name ${VNET_NAME} -n subnet4 --address-prefixes 10.30.4.0/24

    echo "Creating AKS cluster one"
    az aks create --name clusterone --node-count=2 --zones 1 --vnet-subnet-id ${subnet1_id} -g ${AKS_GROUP} --location ${AKS_LOCATION} --enable-managed-identity
    echo "Creating AKS cluster two"
    az aks create --name clustertwo --node-count=2 --zones 1 --vnet-subnet-id ${subnet2_id} -g ${AKS_GROUP} --location ${AKS_LOCATION} --enable-managed-identity
    echo "Creating AKS cluster three"
    az aks create --name clusterthree --node-count=2 --zones 1 --vnet-subnet-id ${subnet3_id} -g ${AKS_GROUP} --location ${AKS_LOCATION} --enable-managed-identity

    az aks get-credentials --name clusterone -g ${AKS_GROUP} --overwrite-existing
    az aks get-credentials --name clustertwo -g ${AKS_GROUP} --overwrite-existing
    az aks get-credentials --name clusterthree -g ${AKS_GROUP} --overwrite-existing
}

helm_install() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    kubectl config use-context clusterone
    kubectl create namespace neo4j
    helm upgrade --install cluster1 neo4j -n neo4j -f ${PROJECT_ROOT}/examples/multi-cluster/cluster-one-values.yaml
    kubectl config use-context clustertwo
    kubectl create namespace neo4j
    helm upgrade --install cluster2 neo4j -n neo4j -f ${PROJECT_ROOT}/examples/multi-cluster/cluster-two-values.yaml
    kubectl config use-context clusterthree
    kubectl create namespace neo4j
    helm upgrade --install cluster3 neo4j -n neo4j -f ${PROJECT_ROOT}/examples/multi-cluster/cluster-three-values.yaml
}

app_gateway() {
    az network public-ip create \
        --resource-group ${AKS_GROUP} \
        --name appGatewayIp \
        --sku Standard \
        --allocation-method static --zone 1

    az network application-gateway create \
      --name ${GATEWAY_NAME} \
      --location ${AKS_LOCATION} \
      --resource-group ${AKS_GROUP} \
      --sku Standard_v2 \
      --public-ip-address appGatewayIp \
      --vnet-name ${VNET_NAME} \
      --subnet subnet4 \
      --servers "10.30.1.101" "10.30.2.101" "10.30.3.101" \
      --frontend-port 7474 \
      --http-settings-port 7474 \
      --http-settings-protocol Http \
      --priority 1

    az network application-gateway frontend-port create \
      --port 7687 \
      --gateway-name ${GATEWAY_NAME} \
      --resource-group ${AKS_GROUP} \
      --name port7687

    az network application-gateway http-listener create \
      --name boltListener \
      --frontend-ip appGatewayFrontendIP \
      --frontend-port port7687 \
      --resource-group ${AKS_GROUP} \
      --gateway-name ${GATEWAY_NAME}

    az network application-gateway rule create \
      --gateway-name ${GATEWAY_NAME} \
      --name rule2 \
      --resource-group ${AKS_GROUP} \
      --http-listener boltListener \
      --priority 2 \
      --address-pool appGatewayBackendPool

}
setup_clusters
helm_install
app_gateway
