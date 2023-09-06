{{- define "neo4j.fullname" -}}
    {{- if .Values.fullnameOverride -}}
        {{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
    {{- else -}}
        {{- if .Values.nameOverride -}}
            {{- $name := default .Chart.Name .Values.nameOverride -}}
            {{- if contains $name .Release.Name -}}
                {{- .Release.Name | trunc 63 | trimSuffix "-" -}}
            {{- else -}}
                {{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
            {{- end -}}
       {{- else -}}
            {{- printf "%s" .Release.Name | trunc 63 | trimSuffix "-" -}}
       {{- end -}}
    {{- end -}}
{{- end -}}
