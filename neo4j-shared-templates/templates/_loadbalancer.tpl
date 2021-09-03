{{- define "neo4j.services.neo4j.defaultSpec" -}}
ClusterIP:
  sessionAffinity: None
NodePort:
  sessionAffinity: None
  externalTrafficPolicy: Local
LoadBalancer:
  sessionAffinity: None
  externalTrafficPolicy: Local
{{- end }}


{{- define "neo4j.services.extraSpec" -}}
{{- if hasKey . "type" }}{{ fail "field 'type' is not supported in Neo4j Helm Chart service.*.spec" }}{{ end }}
{{- if hasKey . "selector" }}{{ fail "field 'selector' is not supported in Neo4j Helm Chart service.*.spec" }}{{ end }}
{{- if hasKey . "ports" }}{{ fail "field 'ports' is not supported in Neo4j Helm Chart service.*.spec" }}{{ end }}
{{- if hasKey . "publishNotReadyAddresses" }}{{ fail "field 'publishNotReadyAddresses' is not supported in Neo4j Helm Chart service.*.spec" }}{{ end }}
{{ toYaml . }}
{{- end }}

{{- define "neo4j.services.neo4j" -}}
{{- $defaultSpec := include "neo4j.services.neo4j.defaultSpec" . | fromYaml }}
{{- if .Values.enabled }}
{{- $spec := get $defaultSpec .Values.spec.type | merge .Values.spec  }}
# Service for applications that need access to neo4j
apiVersion: v1
kind: Service
metadata:
  name: "{{ .Release.Name }}-neo4j"
  namespace: "{{ .Release.Namespace }}"
  labels:
    helm.neo4j.com/neo4j.name: "{{ template "neo4j.name" $ }}"
    app: "{{ template "neo4j.appName" . }}"
    helm.neo4j.com/service: "neo4j"
    {{- include "neo4j.labels" .Values.neo4j | nindent 4 }}
  {{- with .Values.annotations }}
  annotations: {{ toYaml . | nindent 4 }}
  {{- end }}
spec:
  type: "{{ $spec.type | required "service type must be specified" }}"
  {{- omit $spec "type" "ports" "selector" | include "neo4j.services.extraSpec"  | nindent 2 }}
  ports:
    {{- with .Values.ports }}
    {{- if .http.enabled }}
    - protocol: TCP
      port: {{ .http.port | default 7474 }}
      targetPort: 7474
      name: http
    {{- end }}
    {{- if .https.enabled }}
    - protocol: TCP
      port: {{ .https.port | default 7473 }}
      targetPort: 7473
      name: https
    {{- end }}
    {{- if .bolt.enabled }}
    - protocol: TCP
      port: {{ .bolt.port | default 7687 }}
      targetPort: 7687
      name: tcp-bolt
    {{- end }}
    {{ with .backup }}
    {{- if .enabled }}
    - protocol: TCP
      port: {{ .port | default 6362 }}
      targetPort: 6362
      name: tcp-backup
    {{- end }}
    {{- end }}
    {{- end }}
  selector:
    {{- with .Values.selector }}
    {{- . | toYaml | nindent 4 }}
    {{- end }}
{{- end -}}
{{- end -}}
