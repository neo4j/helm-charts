{{/* vim: set filetype=mustache: */}}
{{/*
Convert a neo4j.conf properties text into valid yaml
*/}}
{{- define "neo4j.configYaml" -}}
  {{- regexReplaceAll "(?m)^([^=]*?)=" ( regexReplaceAllLiteral "\\s*(#|dbms\\.jvm\\.additional).*" . "" )  "${1}: " | trim | replace ": true\n" ": 'true'\n" | replace ": true" ": 'true'\n" | replace ": false\n" ": 'false'\n" | replace ": false" ": 'false'\n"  | replace ": yes\n" ": 'yes'\n" | replace ": yes" ": 'yes'\n" | replace ": no" ": 'no'\n" | replace ": no\n" ": 'no'\n" }}
{{- end -}}

{{- define "neo4j.configJvmAdditionalYaml" -}}
  {{- /* This collects together all dbms.jvm.additional entries */}}
  {{- range ( regexFindAll "(?m)^\\s*(dbms\\.jvm\\.additional=).+" . -1 ) }}{{ trim . | replace "dbms.jvm.additional=" "" | trim | nindent 2 }}{{- end }}
{{- end -}}

{{- define "neo4j.appName" -}}
  {{- .Values.neo4j.name | default .Release.Name }}
{{- end -}}

{{- define "neo4j.cluster.server_groups" -}}
  {{- $replicaEnabled := index .Values.config "dbms.mode" | default "" | regexMatch "(?i)READ_REPLICA$" }}
  {{- if $replicaEnabled }}
       {{- "read-replicas" }}
  {{ else }}
       {{- "cores" }}
  {{- end -}}
{{- end -}}


{{/*
If no name is set in `Values.neo4j.name` sets it to release name and modifies Values.neo4j so that the same name is available everywhere
*/}}
{{- define "neo4j.name" -}}
  {{- if not .Values.neo4j.name }}
    {{- $name := .Release.Name }}
    {{- $ignored := set .Values.neo4j "name" $name }}
  {{- end -}}
  {{- .Values.neo4j.name }}
{{- end -}}

{{/*
If no password is set in `Values.neo4j.password` generates a new random password and modifies Values.neo4j so that the same password is available everywhere
*/}}
{{- define "neo4j.password" -}}
  {{- if not .Values.neo4j.password }}
    {{- $password :=  randAlphaNum 14 }}
    {{- $secretName := include "neo4j.appName" . | printf "%s-auth" }}
    {{- $secret := (lookup "v1" "Secret" .Release.Namespace $secretName) }}

    {{- if $secret }}
      {{- $password = index $secret.data "NEO4J_AUTH" | b64dec | trimPrefix "neo4j/" -}}
    {{- end -}}
    {{- $ignored := set .Values.neo4j "password" $password }}
  {{- end -}}
  {{- .Values.neo4j.password }}
{{- end -}}

{{- define "neo4j.checkIfClusterIsPresent" -}}

    {{- $name := .Values.neo4j.name -}}
    {{- $clusterList := list -}}
    {{- range $index,$pod := (lookup "v1" "Pod" .Release.Namespace "").items -}}
        {{- if eq $name (index $pod.metadata.labels "helm.neo4j.com/neo4j.name") -}}
            {{- if eq (index $pod.metadata.labels "helm.neo4j.com/dbms.mode") "CORE" -}}

                {{- $noOfContainers := len (index $pod.status.containerStatuses) -}}
                {{- $noOfReadyContainers := 0 -}}

                {{- range $index,$value := index $pod.status.containerStatuses -}}
                    {{- if $value.ready }}
                        {{- $noOfReadyContainers = add1 $noOfReadyContainers -}}
                    {{- end -}}
                {{- end -}}

                {{/* Number of Ready Containers should be equal to the number of containers in the pod */}}
                {{/* Pod should be in running state */}}
                {{- if and (eq $noOfReadyContainers $noOfContainers) (eq (index $pod.status.phase) "Running") -}}
                    {{- $clusterList = append $clusterList (index $pod.metadata.name) -}}
                {{- end -}}

            {{- end -}}
        {{- end -}}
    {{- end -}}

    {{- if lt (len $clusterList) 3 -}}
        {{ fail "Cannot install Read Replica until a cluster of 3 or more cores is formed" }}
    {{- end -}}

{{- end -}}

{{- define "podSpec.checkLoadBalancerParam" }}
{{- $isLoadBalancerValuePresent := required (include "podSpec.loadBalancer.mustBeSetMessage" .) .Values.podSpec.loadbalancer | regexMatch "(?i)include$|(?i)exclude$" -}}
{{- if not $isLoadBalancerValuePresent }}
{{- include "podSpec.loadBalancer.mustBeSetMessage" . | fail -}}
{{- end }}
{{- end }}

{{- define "podSpec.loadBalancer.mustBeSetMessage" }}

Set podSpec.loadbalancer to one of: "include", "exclude".

E.g. by adding `--set podSpec.loadbalancer=include`

{{ end -}}
