{{- define "neo4j.labels" -}}
    {{- with .labels -}}
        {{- range $name, $value := . }}
{{ $name | quote}}: {{ $value | quote }}
        {{- end -}}
    {{- end -}}
{{- end }}

{{- define "neo4j.annotations" -}}
    {{- with . -}}
        {{- range $name, $value := . }}
{{ $name | quote }}: {{ $value | quote }}
        {{- end -}}
    {{- end -}}
{{- end }}
