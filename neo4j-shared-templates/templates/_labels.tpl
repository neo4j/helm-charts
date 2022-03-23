{{- define "neo4j.labels" -}}
    {{- with .labels -}}
        {{- range $name, $value := . }}
"{{ $name }}": "{{ $value }}"
        {{- end -}}
    {{- end -}}
{{- end }}

{{- define "neo4j.nodeSelector" -}}
{{- if not (empty .) }}
nodeSelector:
    {{- with . -}}
        {{- range $name, $value := . }}
  "{{ $name }}": "{{ $value }}"
        {{- end -}}
    {{- end -}}
{{- end -}}
{{- end }}
