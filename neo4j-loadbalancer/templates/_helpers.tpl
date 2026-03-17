{{- define "neo4j.loadbalancer.name" -}}
  {{- required "neo4j.name is required" .Values.neo4j.name }}
{{- end -}}

{{- define "neo4j.loadbalancer.appName" -}}
  {{- required "neo4j.name is required" .Values.neo4j.name }}
{{- end -}}
