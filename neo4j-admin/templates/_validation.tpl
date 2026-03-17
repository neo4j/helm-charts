{{- define "neo4j.admin.backup.checkIfSecretExistsOrNot" -}}
    {{- if (.Values.backup.secretName | trim) -}}
        {{- if (not .Values.disableLookups) -}}

            {{- include "neo4j.admin.backup.checkIfSecretKeyNameExistsOrNot" . -}}
            {{- $secret := (lookup "v1" "Secret" .Release.Namespace .Values.backup.secretName) }}
            {{- $secretExists := $secret | all }}

            {{- if (not $secretExists) -}}
                {{ fail (printf "Secret %s configured in 'backup.secretName' not found" .Values.backup.secretName) }}
             {{- else if not (hasKey $secret.data .Values.backup.secretKeyName) -}}
                {{ fail (printf "Secret %s must contain key %s" .Values.backup.secretName .Values.backup.secretKeyName) }}
            {{- end -}}
        {{- end -}}
    {{- end -}}
{{- end -}}

{{- define "neo4j.admin.backup.checkAzureStorageAccountName" -}}
    {{- if eq .Values.backup.cloudProvider "azure" }}
        {{- if and (or (empty .Values.backup.secretName) (empty .Values.backup.secretKeyName)) (empty .Values.backup.azureStorageAccountName) -}}
            {{ fail (printf "Both secretName|secretKeyName and azureStorageAccountName key cannot be empty. Please set one of them via --set backup.secretName or --set backup.azureStorageAccountName") }}
        {{- end -}}

        {{- if and (or (.Values.backup.secretName) (.Values.backup.secretKeyName)) (.Values.backup.azureStorageAccountName) -}}
            {{ fail (printf "Both secretName|secretKeyName and azureStorageAccountName key cannot be present. Please set only one of them via --set backup.secretName or --set backup.azureStorageAccountName") }}
        {{- end -}}
    {{- end -}}
{{- end -}}

{{/*check for secretKeyName existence only when secretName is provided*/}}
{{- define "neo4j.admin.backup.checkIfSecretKeyNameExistsOrNot" -}}
   {{- if .Values.backup.secretName -}}
    {{- if kindIs "invalid" .Values.backup.secretKeyName -}}
        {{- fail (printf "Missing secretKeyName !!") -}}
    {{- else if (not (.Values.backup.secretKeyName | trim)) -}}
        {{- fail (printf "Empty secretKeyName") -}}
    {{- end -}}
   {{- end -}}

{{- end -}}

{{/* checks if serviceAccountName is provided or not  when secretName is missing */}}
{{- define "neo4j.admin.backup.checkServiceAccountName" -}}
    {{- if and (empty .Values.serviceAccountName) (empty .Values.backup.secretName) (not (empty .Values.backup.cloudProvider)) -}}
        {{ fail (printf "Please provide either secretName or serviceAccountName. Both cannot be empty. Please set only one of them via --set backup.secretName or --set serviceAccountName") }}
    {{- end -}}
{{- end -}}

{{/* checks if serviceAccountName is provided or not  when secretName is missing */}}
{{- define "neo4j.admin.backup.checkBucketName" -}}
    {{- if or (kindIs "invalid" .Values.backup.aggregate) (not .Values.backup.aggregate.enabled) -}}
        {{- if .Values.backup.cloudProvider -}}
            {{- if empty .Values.backup.bucketName -}}
                {{ fail (printf "Empty bucketName. Please set bucketName via --set backup.bucketName") }}
            {{- end -}}
        {{- end -}}
    {{- end -}}
{{- end -}}

{{- define "neo4j.admin.backup.checkDatabaseIPAndServiceName" -}}

    {{- if or (kindIs "invalid" .Values.backup.aggregate) (not .Values.backup.aggregate.enabled) -}}
        {{- if and (kindIs "invalid" .Values.backup.databaseAdminServiceName) (kindIs "invalid" .Values.backup.databaseAdminServiceIP) (kindIs "invalid" .Values.backup.databaseBackupEndpoints) -}}
            {{- fail (printf "Missing fields. Please set databaseAdminServiceName via --set backup.databaseAdminServiceName or databaseAdminServiceIP via --set backup.databaseAdminServiceIP or databaseBackupEndpoints via --set backup.databaseBackupEndpoints")}}
        {{- end -}}

        {{- if and (empty (.Values.backup.databaseAdminServiceName | trim)) (empty (.Values.backup.databaseAdminServiceIP | trim)) (empty (.Values.backup.databaseBackupEndpoints | trim)) -}}
            {{- fail (printf "Empty fields. Please set databaseAdminServiceName via --set backup.databaseAdminServiceName or databaseAdminServiceIP via --set backup.databaseAdminServiceIP or databaseBackupEndpoints via --set backup.databaseBackupEndpoints")}}
        {{- end -}}

        {{- if or (and (.Values.backup.databaseAdminServiceName | trim) (.Values.backup.databaseAdminServiceIP | trim)) (and (.Values.backup.databaseAdminServiceName | trim) (.Values.backup.databaseBackupEndpoints | trim)) (and (.Values.backup.databaseAdminServiceIP | trim) (.Values.backup.databaseBackupEndpoints | trim)) -}}
            {{- fail (printf "Please set only one of: databaseAdminServiceName via --set backup.databaseAdminServiceName or databaseAdminServiceIP via --set backup.databaseAdminServiceIP or databaseBackupEndpoints via --set backup.databaseBackupEndpoints")}}
        {{- end -}}
    {{- end -}}

{{- end -}}

{{/* Validate and set default timeout for consistency check */}}
{{- define "neo4j.admin.backup.validateAndSetTimeout" -}}
    {{- $timeout := .Values.consistencyCheck.timeout | default "" -}}
    {{- if $timeout -}}
        {{- /* Validate timeout format using regex for Go duration format */ -}}
        {{- /* Supports compound durations like "2h30m" and decimal values like "1.5h" */ -}}
        {{- if not (regexMatch "^([0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$" $timeout) -}}
            {{- fail (printf "Invalid timeout format '%s'. Must be a valid Go duration (e.g., '30m', '1h', '2h30m', '1.5h', '4h')" $timeout) -}}
        {{- end -}}
        {{- /* Return the provided timeout */ -}}
        {{- $timeout -}}
    {{- else -}}
        {{- /* Apply conditional default based on cloudProvider */ -}}
        {{- if .Values.backup.cloudProvider }}30m{{- else }}{{- /* No timeout for local storage */ -}}{{- end -}}
    {{- end -}}
{{- end -}}
