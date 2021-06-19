{{- define "neo4j.volumeClaimTemplateSpec"  -}}
{{/*
This template converts a neo4j volume definition into a VolumeClaimTemplate spec based on the specified mode

The desired behaviour is specified by the 'mode' setting. Only the information in the selected 'mode' is used.

All modes except for "volume" are transformed into a "dynamic" spec and then into a plain VolumeClaimTemplate which is then output.
This is to ensure that there aren't dramatically different code paths (all routes ultimately use the same output path)

If "volume" mode is selected nothing is returned.
*/}}
{{- $ignored := required "Values must be passed in to helm volumeTemplate for use with internal templates" .Values -}}
{{- $ignored = required "Template must be passed in to helm volumeTemplate so that tpl function works" .Template -}}

{{/*
Deep Copy the provided volume object so that we can mutate it safely in this template
*/}}
{{- $volume := deepCopy .volume -}}

{{- $validModes := "share|selector|defaultStorageClass|dynamic|volume|volumeClaimTemplate" -}}
{{- if not ( $volume.mode | regexMatch $validModes ) -}}
  {{- fail ( cat "\nUnknown volume mode:" $volume.mode "\nValid modes are: " $validModes ) -}}
{{- end -}}
{{/*
If defaultStorageClass is chosen overwrite "dynamic" and switch to dynamic mode
*/}}
{{-  if eq $volume.mode "defaultStorageClass"  -}}
  {{- $ignored = set $volume "dynamic" .defaultStorageClass -}}
  {{-  if $volume.dynamic.storageClassName -}}
    {{- fail "If using mode defaultStorageClass then storageClassName should not be set" -}}
  {{- end -}}
  {{- $ignored = set $volume "mode" "dynamic" -}}
{{- end -}}

{{/*
If selector is chosen process the selector template and then overwrite "dynamic" and switch to dynamic mode
*/}}
{{- if eq $volume.mode "selector" -}}
  {{- $ignored = set $volume.selector "selector" ( tpl ( toYaml $volume.selector.selectorTemplate ) . | fromYaml ) -}}
  {{- $ignored = set $volume "dynamic" $volume.selector -}}
  {{- $ignored = set $volume "mode" "dynamic" -}}
{{- end -}}

{{- if eq $volume.mode "dynamic" -}}
    {{- $ignored = set $volume "mode" "volumeClaimTemplate" -}}
    {{- $ignored = dict "requests" $volume.dynamic.requests | set $volume.dynamic "resources" -}}
    {{- $ignored = set $volume "volumeClaimTemplate" ( omit $volume.dynamic "requests" "selectorTemplate" ) -}}
{{- end -}}

{{- if eq $volume.mode "volumeClaimTemplate" -}}
    {{- omit $volume.volumeClaimTemplate "setOwnerAndGroupWritableFilePermissions" | toYaml  -}}
{{- end -}}
{{- end -}}

{{- define "neo4j.volumeSpec" -}}
{{- $ignored := required "Values must be passed in to helm volumeTemplate for use with internal templates" .Values -}}
{{- $ignored = required "Template must be passed in to helm volumeTemplate so that tpl function works" .Template -}}
{{- if eq .volume.mode "volume" -}}
{{ omit .volume.volume "setOwnerAndGroupWritableFilePermissions" | toYaml  }}
{{- end -}}
{{- end -}}

{{- define "neo4j.volumeClaimTemplates" -}}
{{- $neo4jName := include "neo4j.name" . }}
{{- $template := .Template -}}
{{- range $name, $spec := .Values.volumes -}}
{{- if $spec -}}
{{- $volumeClaim := dict "Template" $template "Values" $.Values "volume" $spec | include "neo4j.volumeClaimTemplateSpec" -}}
{{- if $volumeClaim -}}
- metadata:
    name: "{{ $name }}"
  spec: {{- $volumeClaim | nindent 4 -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* This template doesn't output anything unless the mode is "volume" */}}
{{- define "neo4j.volumes" -}}
{{- $template := .Template -}}
{{- range $name, $spec := .Values.volumes -}}
{{- if $spec -}}
{{- $volumeYaml := dict "Template" $template "Values" $.Values "volume" $spec | include "neo4j.volumeSpec" -}}
{{- if $volumeYaml }}
- name: "{{ $name }}"
  {{- $volumeYaml | nindent 2 }}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "neo4j.initChmodContainer" }}
{{- $initChmodScript := include "neo4j.initChmodScript" . }}
{{- if $initChmodScript }}
name: "set-volume-permissions"
image: "{{ template "neo4j.image" . }}"
env:
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
securityContext:
  runAsNonRoot: false
  runAsUser: 0
  runAsGroup: 0
volumeMounts: {{- include "neo4j.volumeMounts" .Values.volumes | nindent 2 }}
command:
  - "bash"
  - "-c"
  - |
    set -o pipefail -o errtrace -o errexit -o nounset
    shopt -s inherit_errexit
    [[ -n "${TRACE:-}" ]] && set -o xtrace
    {{- $initChmodScript | nindent 4 }}
{{- end }}
{{- end }}

{{- define "neo4j.initChmodScript" -}}
{{- $securityContext := .Values.securityContext -}}
{{- range $name, $spec := .Values.volumes -}}
{{- if (index $spec $spec.mode).setOwnerAndGroupWritableFilePermissions -}}
{{- if $securityContext -}}{{- if $securityContext.runAsUser }}

# change owner
chown -R "{{ $securityContext.runAsUser }}" "/{{ $name }}"
{{- end -}}{{- end -}}
{{- if $securityContext -}}{{- if $securityContext.runAsGroup }}

# change group
chgrp -R "{{ $securityContext.runAsGroup }}" "/{{ $name }}"
{{- end -}}{{- end }}

# make group writable
chmod -R g+rwx "/{{ $name }}"
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "neo4j.volumeMounts" -}}
{{- range $name, $spec := . }}
- mountPath: "/{{ $name }}"
  name: "{{ if eq $spec.mode "share" }}{{ $spec.share.name }}{{ if eq $name "data" }}{{ fail "data volume does not support mode: 'share'"}}{{ end }}{{ else }}{{ $name }}{{ end }}"
  subPathExpr: "{{ $name }}{{ if regexMatch "logs|metrics" $name }}/$(POD_NAME){{ end }}"
{{- end -}}
{{- end -}}

{{- define "neo4j.maintenanceVolumeMounts" -}}
{{- range $name, $spec := . }}
{{- if ne $spec.mode "share" | and (ne "data" $name) }}
- mountPath: "/maintenance/{{ $name }}_volume"
  name: "{{ $name }}"
  readOnly: true
{{- end -}}
{{- end -}}
{{- end -}}
