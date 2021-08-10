{{- define "neo4j.services.neo4j.defaultSpec" -}}
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
{{- $spec := merge .Values.spec $defaultSpec }}
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
      port: 7474
      targetPort: 7474
      name: http
    {{- end }}
    {{- if .https.enabled }}
    - protocol: TCP
      port: 7473
      targetPort: 7473
      name: https
    {{- end }}
    {{- if .bolt.enabled }}
    - protocol: TCP
      port: 7687
      targetPort: 7687
      name: tcp-bolt
    {{- end }}
    {{- end }}
  selector:
    app: "{{ template "neo4j.appName" . | required "neo4j.name must be specified" }}"
    {{- with .Values.selector }}
    {{- . | toYaml | nindent 4 }}
    {{- end }}
{{- end -}}
{{- end -}}
