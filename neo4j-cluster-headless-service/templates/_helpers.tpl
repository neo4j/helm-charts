{{- define "neo4j.name" -}}
  {{- required "neo4j.name is required" .Values.neo4j.name }}
{{- end -}}

{{- define "neo4j.appName" -}}
  {{- required "neo4j.name is required" .Values.neo4j.name }}
{{- end -}}

{{- define "neo4j.checkPortMapping" -}}
    {{- $httpPort := .Values.ports.http.port | int | default 7474 -}}
    {{- $httpsPort := .Values.ports.https.port | int | default 7473 -}}
    {{- $boltPort := .Values.ports.bolt.port | int | default 7687 -}}
    {{- $backupPort := .Values.ports.backup.port | int | default 6362 -}}

    {{- if and (eq .Values.ports.http.enabled true) (ne $httpPort 7474) -}}
        {{- include "neo4j.portRemappingFailureMessage" $httpPort -}}
    {{- end -}}
    {{- if and (eq .Values.ports.https.enabled true) (ne $httpsPort 7473) -}}
        {{- include "neo4j.portRemappingFailureMessage" $httpsPort -}}
    {{- end -}}
    {{- if and (eq .Values.ports.bolt.enabled true) (ne $boltPort 7687) -}}
        {{- include "neo4j.portRemappingFailureMessage" $boltPort -}}
    {{- end -}}
    {{- if and (eq .Values.ports.backup.enabled true) (ne $backupPort 6362) -}}
        {{- include "neo4j.portRemappingFailureMessage" $backupPort -}}
    {{- end -}}
{{- end -}}

{{- define "neo4j.portRemappingFailureMessage" -}}
    {{- $message := . | printf "port re-mapping is not allowed in headless service. Please remove custom port %d from values.yaml" -}}
    {{- fail $message -}}
{{- end -}}
