{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
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

{{/* checkNodeSelectorLabels checks if there is any node in the cluster which has nodeSelector labels */}}
{{- define "neo4j.checkNodeSelectorLabels" -}}
    {{- if and (not (empty $.Values.nodeSelector)) (not $.Values.disableLookups) -}}
        {{- $validNodes := 0 -}}
        {{- $numberOfLabelsRequired := len $.Values.nodeSelector -}}
        {{- range $index, $node := (lookup "v1" "Node" .Release.Namespace "").items -}}
            {{- $nodeLabels :=  $node.metadata.labels -}}
            {{- $numberOfLabelsFound := 0 -}}
            {{/* match all the nodeSelector labels with the existing node labels*/}}
            {{- range $name,$value := $.Values.nodeSelector -}}
                {{- if hasKey $nodeLabels $name -}}
                    {{- if eq ($value | toString) (get $nodeLabels $name | toString ) -}}
                        {{- $numberOfLabelsFound = add1 $numberOfLabelsFound -}}
                    {{- end -}}
                {{- end -}}
            {{- end -}}

            {{/* increment valid nodes if the number of labels required are matching with the number of labels found */}}
            {{- if eq $numberOfLabelsFound $numberOfLabelsRequired -}}
                {{- $validNodes = add1 $validNodes -}}
            {{- end -}}
        {{- end -}}

        {{- if eq $validNodes 0 -}}
            {{- fail (print "No node exists in the cluster which has all the below labels (.Values.nodeSelector) \n %s" ($.Values.nodeSelector | join "\n" | toString) ) -}}
        {{- end -}}
    {{- end -}}
{{- end -}}

{{- define "neo4j.tolerations" -}}
{{/* Add tolerations only if .Values.tolerations contains entries */}}
    {{- if . -}}
tolerations:
{{ toYaml . }}
    {{- end -}}
{{- end -}}

{{- define "neo4j.affinity" -}}
    {{- if . -}}
affinity:
{{ toYaml . | indent 1 }}
    {{- end -}}
{{- end -}}

{{- define "neo4j.resourcesAndLimits" -}}
requests:
  ephemeral-storage: {{ .Values.resources.requests.ephemeralStorage | default "4Gi" }}

  {{- if and (not (kindIs "invalid" .Values.resources.requests.cpu)) (not (empty $.Values.resources.requests.cpu)) }}
  cpu: {{ .Values.resources.requests.cpu | default "" }}
  {{- end }}

  {{- if and (not (kindIs "invalid" .Values.resources.requests.memory)) (not (empty $.Values.resources.requests.memory)) }}
  memory: {{ .Values.resources.requests.memory | default "" }}
  {{- end }}
limits:
  ephemeral-storage: {{ .Values.resources.limits.ephemeralStorage | default "5Gi" }}

  {{- if and (not (kindIs "invalid" .Values.resources.limits.cpu)) (not (empty $.Values.resources.limits.cpu)) }}
  cpu: {{ .Values.resources.limits.cpu | default "" }}
  {{- end }}

  {{- if and (not (kindIs "invalid" .Values.resources.limits.memory)) (not (empty $.Values.resources.limits.memory)) }}
  memory: {{ .Values.resources.limits.memory | default "" }}
  {{- end }}

{{- end -}}
