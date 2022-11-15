# Deploy Neo4j with enterprise Bloom and GDS plugins

This example demonstrates deploying a standalone Neo4j server with Bloom and GDS plugins, both with and without license keys.

# Deploy Neo4j and GDS without license
The script will use the following Helm values
```yaml
neo4j:
  name: gds-no-license
  acceptLicenseAgreement: "yes"
  edition: enterprise
volumes:
  data:
    mode: defaultStorageClass
env:
  NEO4J_PLUGINS: '["graph-data-science"]'
config:
  dbms.security.procedures.unrestricted: "gds.*,apoc.*"
```

To install using these values, run the script:
```shell
./install-gds-no-license.sh
```

# Deploy Neo4j, GDS and Bloom with license files
The script will use the following Helm values
```yaml
neo4j:
  name: licenses
  acceptLicenseAgreement: "yes"
  edition: enterprise
volumes:
  data:
    mode: defaultStorageClass
  licenses:
    mode: volume
    volume:
      secret:
        secretName: gds-bloom-license
        items:
          - key: gds.license
            path: gds.license
          - key: bloom.license
            path: bloom.license
env:
  NEO4J_PLUGINS: '["graph-data-science", "bloom"]'
config:
  gds.enterprise.license_file: "/licenses/gds.license"
  dbms.security.procedures.unrestricted: "gds.*,apoc.*,bloom.*"
  server.unmanaged_extension_classes: "com.neo4j.bloom.server=/bloom,semantics.extension=/rdf"
  dbms.security.http_auth_allowlist: "/,/browser.*,/bloom.*"
  dbms.bloom.license_file: "/licenses/bloom.license"
```
To install using these values, run the script:

**N.B The example requires the license files for Bloom and GDS and assumes the files are named gds.license and bloom.license** 
```shell
./install-gds-bloom-with-license.sh /path/to/gds.license /path/to/bloom.license
```
