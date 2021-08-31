{{- define "neo4j.labels" -}}
{{- with .labels }}
{{- range $name, $value := . }}
"{{ $name }}": "{{ $value }}"
{{ end }}
{{- end }}
{{- end -}}
