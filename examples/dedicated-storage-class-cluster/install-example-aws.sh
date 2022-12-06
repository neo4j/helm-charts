#!/usr/bin/env bash
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly RELEASE_NAME=dedicated-storage-class

helm_install() {
    if ! kubectl get daemonset ebs-csi-node -n kube-system &> /dev/null; then
        echo "WARNING: EBS CSI Driver not found, this example will not work."
        echo "See https://docs.aws.amazon.com/eks/latest/userguide/ebs-csi.html for instructions to install driver"
    fi
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    kubectl create secret generic neo4j-auth --from-literal=NEO4J_AUTH=neo4j/password123
    kubectl apply -f examples/dedicated-storage-class-cluster/aws-storage-class.yaml
    for i in {1..3}; do
        helm install "${RELEASE_NAME}-${i}" neo4j -fexamples/dedicated-storage-class-cluster/dedicated-storage-class.yaml
    done
}

helm_install
