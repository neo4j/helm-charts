{{- define "neo4j.defaultChartImage" -}}
{{- $isEnterprise := required "neo4j.edition must be specified" .Values.neo4j.edition | regexMatch "(?i)enterprise" -}}
neo4j:{{ .Chart.AppVersion }}{{ if $isEnterprise }}-enterprise{{ end }}
{{- end -}}


{{- define "neo4j.image" -}}
{{- template "neo4j.checkLicenseAgreement" . -}}
{{- $image := include "neo4j.defaultChartImage" . -}}
{{/* Allow override if a custom image has been specified */}}
{{- if .Values.image -}}
  {{- if .Values.image.customImage -}}
    {{- $image = .Values.image.customImage -}}
  {{- end -}}
{{- end -}}
{{ $image }}
{{- end -}}
