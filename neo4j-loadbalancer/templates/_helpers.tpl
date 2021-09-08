{{- define "neo4j.name" -}}
  {{- required "neo4j.name is required" .Values.neo4j.name }}
{{- end -}}

{{- define "neo4j.appName" -}}
  {{- required "neo4j.name is required" .Values.neo4j.name }}
{{- end -}}
