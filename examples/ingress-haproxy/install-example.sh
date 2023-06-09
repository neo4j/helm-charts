#!/usr/bin/env bash
#readonly PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
readonly PROJECT_ROOT="$(builtin cd ../../; pwd)"
readonly SOURCE_DIR="${PROJECT_ROOT}/examples/ingress-haproxy"

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
    sed "s/MYIP/$external_ip/g" "${SOURCE_DIR}"/neo4j-db-ingress-subchart/values.yaml | \
      helm upgrade -i dbingress "${SOURCE_DIR}"/neo4j-db-ingress-subchart -f -

}

helm_install
