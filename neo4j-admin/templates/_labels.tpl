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

{{- define "neo4j.nodeSelector" -}}
{{- if and (not (kindIs "invalid" .Values.nodeSelector) ) (not (empty .Values.nodeSelector) ) }}
{{ printf "nodeSelector" | indent 10 }}: {{ .Values.nodeSelector | toYaml | nindent 12 }}
{{- end }}
{{- end }}
