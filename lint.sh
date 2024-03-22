#!/bin/sh
helm lint neo4j-pod-002 ./neo4j --namespace query-engine-002 -f values.yaml