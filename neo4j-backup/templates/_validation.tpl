{{- define "neo4j.backup.checkIfSecretExistsOrNot" -}}
    {{- if (.Values.backup.secretName | trim) -}}
        {{- if (not .Values.disableLookups) -}}

            {{- include "neo4j.backup.checkIfSecretKeyNameExistsOrNot" . -}}
            {{- $secret := (lookup "v1" "Secret" .Release.Namespace .Values.backup.secretName) }}
            {{- $secretExists := $secret | all }}

            {{- if (not $secretExists) -}}
                {{ fail (printf "Secret %s configured in 'backup.secretname' not found" .Values.backup.secretName) }}
             {{- else if not (hasKey $secret.data .Values.backup.secretKeyName) -}}
                {{ fail (printf "Secret %s must contain key %s" .Values.backup.secretName .Values.backup.secretKeyName) }}
            {{- end -}}
        {{- end -}}
    {{- else -}}
        {{- fail (printf "Missing secretName. Set it via --set backup.secretName")  -}}
    {{- end -}}
{{- end -}}

{{- define "neo4j.backup.checkIfSecretKeyNameExistsOrNot" -}}

        {{- if kindIs "invalid" .Values.backup.secretKeyName -}}
            {{- fail (printf "Missing secretKeyName !!") -}}
        {{- else if (not (.Values.backup.secretKeyName | trim)) -}}
            {{- fail (printf "Empty secretKeyName") -}}
        {{- end -}}

{{- end -}}

{{- define "neo4j.backup.checkDatabaseIPAndServiceName" -}}

    {{- if and (kindIs "invalid" .Values.backup.databaseAdminServiceName) (kindIs "invalid" .Values.backup.databaseAdminServiceIP) -}}
        {{- fail (printf "Missing fields. Please set databaseAdminServiceName via --set backup.databaseAdminServiceName or databaseAdminServiceIP via --set backup.databaseAdminServiceIP")}}
    {{- end -}}

    {{- if and (empty (.Values.backup.databaseAdminServiceName | trim)) (empty (.Values.backup.databaseAdminServiceIP | trim)) -}}
        {{- fail (printf "Empty fields. Please set databaseAdminServiceName via --set backup.databaseAdminServiceName or databaseAdminServiceIP via --set backup.databaseAdminServiceIP")}}
    {{- end -}}

        {{- if and (.Values.backup.databaseAdminServiceName | trim) (.Values.backup.databaseAdminServiceIP | trim) -}}
        {{- fail (printf "Please set databaseAdminServiceName via --set backup.databaseAdminServiceName or databaseAdminServiceIP via --set backup.databaseAdminServiceIP. Cannot use both")}}
    {{- end -}}

{{- end -}}
