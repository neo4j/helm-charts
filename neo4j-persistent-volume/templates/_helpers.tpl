{{- define "neo4j.appName" -}}
  {{ required "neo4j.name is required" .Values.neo4j.name }}
{{- end -}}
