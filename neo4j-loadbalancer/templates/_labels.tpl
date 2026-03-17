{{- define "neo4j.loadbalancer.labels" -}}
    {{- with .labels -}}
        {{- range $name, $value := . }}
{{ $name | quote}}: {{ $value | quote }}
        {{- end -}}
    {{- end -}}
{{- end }}

{{- define "neo4j.loadbalancer.annotations" -}}
    {{- with . -}}
        {{- range $name, $value := . }}
{{ $name | quote }}: {{ $value | quote }}
        {{- end -}}
    {{- end -}}
{{- end }}
