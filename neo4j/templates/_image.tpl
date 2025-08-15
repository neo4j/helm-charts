{{- define "neo4j.defaultChartImage" -}}
{{- $isEnterprise := required "neo4j.edition must be specified" .Values.neo4j.edition | regexMatch "(?i)enterprise" -}}
 {{- $imageName := "neo4j:" -}}
 {{/* .Chart.AppVersion is set to "-" for headless and loadbalancer service*/}}
 {{- if eq $.Chart.AppVersion "-" -}}
    {{- $imageName = printf "%s%s" $imageName $.Chart.Version -}}
 {{- else -}}
    {{- $imageName = printf "%s%s" $imageName $.Chart.AppVersion -}}
 {{- end -}}
 {{- if $isEnterprise -}}
    {{- $imageName = printf "%s%s" $imageName "-enterprise" -}}
 {{- end -}}
 {{- $imageName -}}
{{- end -}}


{{- define "neo4j.image" -}}
{{- template "neo4j.checkLicenseAgreement" . -}}

{{/* Validation: cannot use both customImage and separated fields */}}
{{- if and .Values.image.customImage (or .Values.image.registry .Values.image.repository .Values.image.tag) -}}
{{- fail "Cannot use both image.customImage and separated image fields (image.registry, image.repository, image.tag). Choose only one method." -}}
{{- end -}}

{{/* Validation: repository must be set if using separated fields */}}
{{- if and (or .Values.image.registry .Values.image.tag .Values.image.repository) (eq (trim (.Values.image.repository | default "")) "") -}}
{{- fail "image.repository must be set if using separated image fields" -}}
{{- end -}}

{{- $default_image := include "neo4j.defaultChartImage" . -}}
{{- $image := "" -}}

{{- if .Values.image.customImage -}}
  {{- $image = .Values.image.customImage -}}
{{- else -}}
  {{- $registry := .Values.image.registry | default "" -}}
  {{- $repository := .Values.image.repository | default "neo4j" -}}
  {{- $tag := .Values.image.tag | default (regexReplaceAll "^neo4j:" $default_image "") -}}
  {{- $image = printf "%s%s:%s" (ternary (printf "%s/" $registry) "" (ne $registry "")) $repository $tag -}}
{{- end -}}

{{ $image }}
{{- end -}}

{{- define "cleanup.image" -}}
{{- with .Values.services.neo4j.cleanup.image -}}
    {{- $registryName := .registry -}}
    {{- $repositoryName := .repository -}}
    {{- $separator := ":" -}}
    {{- $termination := printf "%s.%s" $.Capabilities.KubeVersion.Major (regexReplaceAll "\\D+" $.Capabilities.KubeVersion.Minor "") -}}
    {{- if not (empty (.tag | trim)) -}}
        {{- $termination := .tag | toString -}}
    {{- end -}}
    {{- if .digest }}
        {{- $separator = "@" -}}
        {{- $termination = .digest | toString -}}
    {{- end -}}
    {{- printf "%s/%s%s%s" $registryName $repositoryName $separator $termination -}}
{{- end -}}
{{- end -}}
