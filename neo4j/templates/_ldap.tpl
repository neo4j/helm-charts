{{- define "neo4j.ldap.ldapPasswordFromSecret" -}}
    {{- if and (.Values.ldapPasswordFromSecret | trim) (not .Values.disableLookups) -}}
            {{- $secret := (lookup "v1" "Secret" .Release.Namespace .Values.ldapPasswordFromSecret) }}
            {{- $secretExists := $secret | all }}
            {{- if not $secretExists -}}
                {{ fail (printf "Secret %s configured in 'neo4j.ldapPasswordFromSecret' not found" .Values.ldapPasswordFromSecret) }}
            {{- else if not (hasKey $secret.data "LDAP_PASS") -}}
                {{ fail (printf "Secret %s must contain key LDAP_PASS" .Values.ldapPasswordFromSecret) }}
            {{- else -}}
                {{ get $secret.data "LDAP_PASS" |b64dec }}
            {{- end -}}
    {{- end -}}
{{- end -}}
