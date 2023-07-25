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
