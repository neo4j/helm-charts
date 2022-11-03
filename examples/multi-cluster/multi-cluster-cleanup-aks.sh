#!/usr/bin/env bash
readonly AKS_GROUP=multiClusterGroup
cleanup() {
    az aks delete -y --name clusterone -g ${AKS_GROUP}
    az aks delete -y --name clustertwo -g ${AKS_GROUP}
    az aks delete -y --name clusterthree -g ${AKS_GROUP}
    az network application-gateway delete --name multiClusterGateway -g ${AKS_GROUP}
    az network vnet delete --name multiClusterVnet  --resource-group ${AKS_GROUP}
    az network public-ip delete -n appGatewayIp -g ${AKS_GROUP}
    az group delete -g ${AKS_GROUP} -y
}

cleanup
