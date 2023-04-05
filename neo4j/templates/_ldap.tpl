{{- define "neo4j.ldapPasswordFromSecretExistsOrNot" -}}
    {{- template "neo4j.ldapPasswordMountPath" . -}}
    {{- if (.Values.ldapPasswordFromSecret | trim) -}}
        {{- if (not .Values.disableLookups) -}}
            {{- $secret := (lookup "v1" "Secret" .Release.Namespace .Values.ldapPasswordFromSecret) }}
            {{- $secretExists := $secret | all }}

            {{- if (not $secretExists) -}}
                {{ fail (printf "Secret %s configured in 'ldapPasswordFromSecret' not found" .Values.ldapPasswordFromSecret) }}
            {{- else if not (hasKey $secret.data "LDAP_PASS") -}}
                {{ fail (printf "Secret %s must contain key LDAP_PASS" .Values.ldapPasswordFromSecret) }}
            {{- end -}}
        {{- end -}}
        {{- true -}}
    {{- end -}}
{{- end -}}

{{/* checks if ldapPasswordMountPath is set or not when ldapPasswordFromSecret is defined */}}
{{- define "neo4j.ldapPasswordMountPath" -}}
    {{- if (.Values.ldapPasswordFromSecret | trim) -}}
            {{- if not (.Values.ldapPasswordMountPath | trim) -}}
                {{ fail (printf "Please define 'ldapPasswordMountPath'") }}
            {{- end -}}
    {{- else -}}
            {{- if (.Values.ldapPasswordMountPath | trim) -}}
                {{ fail (printf "Please define 'ldapPasswordFromSecret'") }}
            {{- end -}}
    {{- end -}}
{{- end -}}

{{/* checks if ldapPasswordMountPath is set or not when ldapPasswordFromSecret is defined */}}
{{- define "neo4j.ldapVolumeMount" -}}
    {{- if and (.Values.ldapPasswordFromSecret | trim) (.Values.ldapPasswordMountPath | trim) }}
- mountPath: "{{ .Values.ldapPasswordMountPath }}"
  readOnly: true
  name: neo4j-ldap-password
    {{- end }}
{{- end -}}

{{- define "neo4j.ldapVolume" -}}
    {{- if and (.Values.ldapPasswordFromSecret | trim) (.Values.ldapPasswordMountPath | trim) }}
- name: neo4j-ldap-password
  secret:
    secretName: "{{- .Values.ldapPasswordFromSecret -}}"
    {{- end }}
{{- end -}}
