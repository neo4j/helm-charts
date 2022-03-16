{{- define "neo4j.labels" -}}
{{- with .labels -}}
{{- range $name, $value := . }}
"{{ $name }}": "{{ $value }}"
{{- end -}}
{{- end -}}
{{- end }}

{{- define "neo4j.nodeSelector" -}}
{{- if and (not (empty .)) (ne (len ( . | join "" | trim)) 0) }}
nodeSelector:
    {{- with . -}}
        {{- range $name, $value := . }}
  "{{ $name }}": "{{ $value }}"
        {{- end -}}
    {{- end -}}
{{- end -}}
{{- end }}
