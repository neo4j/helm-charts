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
{{ toYaml . }}
{{- end }}

{{/*{{- define "neo4j.services.neo4j" -}}*/}}
{{/*{{- $defaultSpec := include "neo4j.services.neo4j.defaultSpec" . | fromYaml }}*/}}
{{/*{{- if .Values.enabled }}*/}}
{{/*{{- $spec := get $defaultSpec .Values.spec.type | merge .Values.spec  }}*/}}
{{/*# Service for applications that need access to neo4j*/}}
{{/*apiVersion: v1*/}}
{{/*kind: Service*/}}
{{/*metadata:*/}}
{{/*  name: "{{ .Release.Name }}-neo4j"*/}}
{{/*  namespace: "{{ .Release.Namespace }}"*/}}
{{/*  labels:*/}}
{{/*    helm.neo4j.com/neo4j.name: "{{ template "neo4j.name" $ }}"*/}}
{{/*    app: "{{ template "neo4j.name" . }}"*/}}
{{/*    helm.neo4j.com/service: "neo4j"*/}}
{{/*    {{- include "neo4j.labels" .Values.neo4j | indent 4 }}*/}}
{{/*  {{- with .Values.annotations }}*/}}
{{/*  annotations: {{ toYaml . | nindent 4 }}*/}}
{{/*  {{- end }}*/}}
{{/*spec:*/}}
{{/*  {{- if $.Values.multiCluster }}*/}}
{{/*  publishNotReadyAddresses: true*/}}
{{/*  {{- end }}*/}}
{{/*  type: "{{ $spec.type | required "service type must be specified" }}"*/}}
{{/*  {{- omit $spec "type" "ports" "selector" | include "neo4j.services.extraSpec"  | nindent 2 }}*/}}
{{/*  ports:*/}}
{{/*    {{- with .Values.ports }}*/}}
{{/*    {{- if .http.enabled }}*/}}
{{/*    - protocol: TCP*/}}
{{/*      port: {{ .http.port | default 7474 }}*/}}
{{/*      targetPort: 7474*/}}
{{/*      name: http*/}}
{{/*    {{- end }}*/}}
{{/*    {{- if .https.enabled }}*/}}
{{/*    - protocol: TCP*/}}
{{/*      port: {{ .https.port | default 7473 }}*/}}
{{/*      targetPort: 7473*/}}
{{/*      name: https*/}}
{{/*    {{- end }}*/}}
{{/*    {{- if .bolt.enabled }}*/}}
{{/*    - protocol: TCP*/}}
{{/*      port: {{ .bolt.port | default 7687 }}*/}}
{{/*      targetPort: 7687*/}}
{{/*      name: tcp-bolt*/}}
{{/*    {{- end }}*/}}
{{/*    {{ with .backup }}*/}}
{{/*    {{- if .enabled }}*/}}
{{/*    - protocol: TCP*/}}
{{/*      port: {{ .port | default 6362 }}*/}}
{{/*      targetPort: 6362*/}}
{{/*      name: tcp-backup*/}}
{{/*    {{- end }}*/}}
{{/*    {{- end }}*/}}
{{/*    {{- end }}*/}}
{{/*    */}}{{/* this condition opens internal ports only when multi-k8s-cluster is enabled */}}
{{/*    {{- if .Values.multiCluster }}*/}}
{{/*    - name: tcp-boltrouting*/}}
{{/*      protocol: TCP*/}}
{{/*      port: 7688*/}}
{{/*      targetPort: 7688*/}}
{{/*    - name: tcp-discovery*/}}
{{/*      protocol: TCP*/}}
{{/*      port: 5000*/}}
{{/*      targetPort: 5000*/}}
{{/*    - name: tcp-raft*/}}
{{/*      protocol: TCP*/}}
{{/*      port: 7000*/}}
{{/*      targetPort: 7000*/}}
{{/*    - name: tcp-tx*/}}
{{/*      protocol: TCP*/}}
{{/*      port: 6000*/}}
{{/*      targetPort: 6000*/}}
{{/*    {{- end }}*/}}
{{/*  selector:*/}}
{{/*    {{- with .Values.selector }}*/}}
{{/*    {{- . | toYaml | nindent 4 }}*/}}
{{/*    {{- end }}*/}}
{{/*{{- end -}}*/}}
{{/*{{- end -}}*/}}
