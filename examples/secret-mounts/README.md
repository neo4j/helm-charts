# Secret Mounts for Neo4j Helm Chart

This feature allows to securely mount Kubernetes secrets into the Neo4j pod, particularly useful for seedURI operations and other configurations that require external credentials.

## Overview

The `secretMounts` configuration provides a secure way to mount cloud credentials, certificates, or other secrets without embedding them directly in `values.yaml`. This is especially useful for:

- **seedURI operations**: Mount S3, GCS, or Azure credentials for database restoration from cloud storage
- **TLS certificates**: Mount custom certificates for secure connections
- **Custom configurations**: Mount any secret data required by your Neo4j deployment

## Configuration

Add the `secretMounts` section to your values.yaml:

```yaml
secretMounts:
  mount-name:
    secretName: "kubernetes-secret-name"
    mountPath: "/path/in/container"
    # Optional: specify individual keys to mount
    items:
      - key: "secret-key"
        path: "filename"
    # Optional: set file permissions (default: 0644)
    defaultMode: 0600
```

## Examples

### S3 Credentials for seedURI Restoration

This is the primary use case, mounting S3 credentials for seedURI operations:

```yaml
neo4j:
  name: "my-neo4j"
  edition: "enterprise"

secretMounts:
  s3-credentials:
    secretName: "cloud-s3-creds"
    mountPath: "/var/secrets/s3"
    items:
      - key: "access-key-id"
        path: "access-key"
      - key: "secret-access-key" 
        path: "secret-key"
      - key: "endpoint"
        path: "endpoint"
    defaultMode: 0600

volumes:
  data:
    mode: "defaultStorageClass"
```

**Create the secret:**
```bash
kubectl create secret generic cloud-s3-creds \
  --from-literal=access-key-id="your-access-key" \
  --from-literal=secret-access-key="your-secret-key" \
  --from-literal=endpoint="https://your-cloud-endpoint"
```

### Multiple Secret Mounts

You can mount multiple secrets:

```yaml
secretMounts:
  s3-credentials:
    secretName: "s3-creds"
    mountPath: "/var/secrets/s3"
    defaultMode: 0600
  
  tls-certificates:
    secretName: "custom-certificates"
    mountPath: "/var/secrets/certs"
    
  custom-config:
    secretName: "app-config"
    mountPath: "/var/secrets/config"
    items:
      - key: "config.json"
        path: "app-config.json"
```

## Validation

The Helm chart includes validation to ensure:

- `secretName` and `mountPath` are required for each mount
- When `items` is specified, each item must have both `key` and `path`
- Secrets exist in the cluster (when `disableLookups` is false)
- Specified keys exist in the secrets

## Security Considerations

- All mounted secrets are read-only
- Use `defaultMode` to set appropriate file permissions (recommended: 0600 for sensitive data)
- The Neo4j process runs as user 7474, ensure the mounted files are accessible
- Consider using Kubernetes RBAC to restrict access to secrets
