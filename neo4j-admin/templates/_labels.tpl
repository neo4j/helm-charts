{{- define "neo4j.admin.labels" -}}
    {{- with . -}}
        {{- range $name, $value := . }}
{{ $name }}: "{{ $value }}"
        {{- end -}}
    {{- end -}}
{{- end }}

{{- define "neo4j.admin.annotations" -}}
    {{- with . -}}
        {{- range $name, $value := . }}
{{ $name }}: "{{ $value }}"
        {{- end -}}
    {{- end -}}
{{- end }}

{{- define "neo4j.admin.nodeSelector" -}}
{{- if and (not (kindIs "invalid" .Values.nodeSelector) ) (not (empty .Values.nodeSelector) ) }}
{{ printf "nodeSelector" | indent 10 }}: {{ .Values.nodeSelector | toYaml | nindent 12 }}
{{- end }}
{{- end }}
