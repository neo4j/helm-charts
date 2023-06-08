#!/usr/bin/env bash
#readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"

readonly RELEASE_NAME=ingress-haproxy

helm_install() {
    pushd "${PROJECT_ROOT}" > /dev/null || exit
    kubectl create namespace neo4j || true
    kubectl config set-context --current --namespace=neo4j
    helm upgrade --install ingress-nginx ingress-nginx \
      --repo https://kubernetes.github.io/ingress-nginx \
      --namespace ingress-nginx --create-namespace

    external_ip=""
    while [ -z $external_ip ]; do
        echo "Waiting for end point..."
        external_ip=$(kubectl get svc ingress-nginx-controller -n ingress-nginx --template="{{range .status.loadBalancer.ingress}}{{.ip}}{{end}}")
        [ -z "$external_ip" ] && sleep 5
    done
    kubectl create secret generic test-auth --from-literal=NEO4J_AUTH=neo4j/password123
    kubectl apply -f  "${PROJECT_ROOT}/examples/ingress-haproxy/haproxy.yaml"
    kubectl get service -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
    kubectl create ingress $RELEASE_NAME --class=nginx \
      --rule="${external_ip}.sslip.io/*=neo4j-haproxy:8080"
    for i in {1..3}; do
        helm upgrade --install "${RELEASE_NAME}-${i}" neo4j/neo4j -f examples/ingress-haproxy/ingress-haproxy.yaml
    done
}

helm_install
