{{- define "neo4j.labels" -}}
    {{- with . -}}
        {{- range $name, $value := . }}
{{ $name }}: "{{ $value }}"
        {{- end -}}
    {{- end -}}
{{- end }}

{{- define "neo4j.annotations" -}}
    {{- with . -}}
        {{- range $name, $value := . }}
{{ $name }}: "{{ $value }}"
        {{- end -}}
    {{- end -}}
{{- end }}
