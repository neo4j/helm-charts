{{/* vim: set filetype=mustache: */}}

{{/*
Validate secretMounts configuration
*/}}
{{- define "neo4j.secretMounts.validation" -}}
    {{- if .Values.secretMounts -}}
        {{- template "neo4j.secretMounts.checkRequiredFields" . -}}
        {{- template "neo4j.secretMounts.checkSecretExistence" . -}}
    {{- end -}}
{{- end -}}

{{/*
Check for required fields in secretMounts configuration
*/}}
{{- define "neo4j.secretMounts.checkRequiredFields" -}}
    {{- $errors := list -}}
    {{- range $mountName, $mountConfig := .Values.secretMounts -}}
        {{- if not $mountConfig.secretName -}}
            {{- $errors = append $errors (printf "secretMounts.%s.secretName is required" $mountName) -}}
        {{- end -}}
        {{- if not $mountConfig.mountPath -}}
            {{- $errors = append $errors (printf "secretMounts.%s.mountPath is required" $mountName) -}}
        {{- end -}}
        {{- if and $mountConfig.items (not (kindIs "slice" $mountConfig.items)) -}}
            {{- $errors = append $errors (printf "secretMounts.%s.items must be a list" $mountName) -}}
        {{- end -}}
        {{- range $item := ($mountConfig.items | default list) -}}
            {{- if not $item.key -}}
                {{- $errors = append $errors (printf "secretMounts.%s.items[].key is required" $mountName) -}}
            {{- end -}}
            {{- if not $item.path -}}
                {{- $errors = append $errors (printf "secretMounts.%s.items[].path is required" $mountName) -}}
            {{- end -}}
        {{- end -}}
    {{- end -}}
    
    {{- $errors = compact $errors -}}
    {{- if gt (len $errors) 0 -}}
        {{- fail (printf "secretMounts validation failed: %s" ($errors | join ", ")) -}}
    {{- end -}}
{{- end -}}

{{/*
Check if secrets exist (only when disableLookups is false)
*/}}
{{- define "neo4j.secretMounts.checkSecretExistence" -}}
    {{- if not .Values.disableLookups -}}
        {{- $errors := list -}}
        {{- range $mountName, $mountConfig := .Values.secretMounts -}}
            {{- $secret := (lookup "v1" "Secret" $.Release.Namespace $mountConfig.secretName) -}}
            {{- $secretExists := $secret | all -}}
            
            {{- if not $secretExists -}}
                {{- $errors = append $errors (printf "Secret '%s' configured in secretMounts.%s.secretName not found" $mountConfig.secretName $mountName) -}}
            {{- else -}}
                {{/* Check if specified keys exist in the secret */}}
                {{- range $item := ($mountConfig.items | default list) -}}
                    {{- if not (hasKey $secret.data $item.key) -}}
                        {{- $errors = append $errors (printf "Secret '%s' does not contain key '%s' required by secretMounts.%s" $mountConfig.secretName $item.key $mountName) -}}
                    {{- end -}}
                {{- end -}}
            {{- end -}}
        {{- end -}}
        
        {{- $errors = compact $errors -}}
        {{- if gt (len $errors) 0 -}}
            {{- fail (printf "secretMounts validation failed: %s" ($errors | join ", ")) -}}
        {{- end -}}
    {{- end -}}
{{- end -}}

{{/*
Generate volume mounts for secretMounts
*/}}
{{- define "neo4j.secretMounts.volumeMounts" -}}
    {{- range $mountName, $mountConfig := .Values.secretMounts }}
- name: {{ printf "secret-mount-%s" $mountName | quote }}
  mountPath: {{ $mountConfig.mountPath | quote }}
  readOnly: true
    {{- end }}
{{- end -}}

{{/*
Generate volumes for secretMounts
*/}}
{{- define "neo4j.secretMounts.volumes" -}}
    {{- range $mountName, $mountConfig := .Values.secretMounts }}
- name: {{ printf "secret-mount-%s" $mountName | quote }}
  secret:
    secretName: {{ $mountConfig.secretName | quote }}
    {{- if $mountConfig.defaultMode }}
    defaultMode: {{ $mountConfig.defaultMode }}
    {{- end }}
    {{- if $mountConfig.items }}
    items:
    {{- range $item := $mountConfig.items }}
    - key: {{ $item.key | quote }}
      path: {{ $item.path | quote }}
      {{- if $item.mode }}
      mode: {{ $item.mode }}
      {{- end }}
    {{- end }}
    {{- end }}
    {{- end }}
{{- end -}}
