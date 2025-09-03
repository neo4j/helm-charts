#!/bin/bash

# Script to create Kubernetes secret for S3 credentials
# Usage: ./create-s3-secret.sh [secret-name] [namespace]

set -e

SECRET_NAME="${1:-s3-credentials}"
NAMESPACE="${2:-default}"

echo "Creating Kubernetes secret for S3 credentials..."
echo "Secret name: $SECRET_NAME"
echo "Namespace: $NAMESPACE"
echo

# Prompt for credentials
read -p "Enter S3 Access Key ID: " ACCESS_KEY_ID
read -s -p "Enter S3 Secret Access Key: " SECRET_ACCESS_KEY
echo
read -p "Enter S3 Endpoint (e.g., https://s3.example.com): " ENDPOINT
read -p "Enter S3 Region (e.g., us-east-1): " REGION

echo
echo "Creating secret..."

kubectl create secret generic "$SECRET_NAME" \
  --namespace="$NAMESPACE" \
  --from-literal=access-key-id="$ACCESS_KEY_ID" \
  --from-literal=secret-access-key="$SECRET_ACCESS_KEY" \
  --from-literal=endpoint="$ENDPOINT" \
  --from-literal=region="$REGION"

echo "Secret '$SECRET_NAME' created successfully in namespace '$NAMESPACE'"
echo
echo "You can now use this secret in your values.yaml:"
echo "secretMounts:"
echo "  s3-credentials:"
echo "    secretName: \"$SECRET_NAME\""
echo "    mountPath: \"/var/secrets/s3\""
echo "    items:"
echo "      - key: \"access-key-id\""
echo "        path: \"access-key\""
echo "      - key: \"secret-access-key\""
echo "        path: \"secret-key\""
echo "      - key: \"endpoint\""
echo "        path: \"endpoint\""
echo "      - key: \"region\""
echo "        path: \"region\""
echo "    defaultMode: 0600"
