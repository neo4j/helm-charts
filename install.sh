#!/bin/sh
helm install neo4j-pod-002 ./neo4j --namespace query-engine-002 --create-namespace -f values.yaml