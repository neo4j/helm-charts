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
        {{- if eq $name (index $pod.metadata.labels "helm.neo4j.com/neo4j.name" | toString) -}}
            {{- if eq (index $pod.metadata.labels "helm.neo4j.com/dbms.mode" | toString) "CORE" -}}

                {{- $noOfContainers := len (index $pod.status.containerStatuses) -}}
                {{- $noOfReadyContainers := 0 -}}

                {{- range $index,$value := index $pod.status.containerStatuses -}}
                    {{- if $value.ready }}
                        {{- $noOfReadyContainers = add1 $noOfReadyContainers -}}
                    {{- end -}}
                {{- end -}}

                {{/* Number of Ready Containers should be equal to the number of containers in the pod */}}
                {{/* Pod should be in running state */}}
                {{- if and (eq $noOfReadyContainers $noOfContainers) (eq (index $pod.status.phase | toString) "Running") -}}
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

{{- define "neo4j.checkResources" -}}

    {{- template "neo4j.resources.checkForEmptyResources" . -}}

    {{- template "neo4j.resources.evaluateCPU" . -}}

    {{- template "neo4j.resources.evaluateMemory" . -}}

{{- end -}}

{{/* checks if the resources are empty or not */}}
{{- define "neo4j.resources.checkForEmptyResources" -}}

    {{/* check for missing cpu and memory values */}}
    {{- if and (kindIs "invalid" .Values.neo4j.resources) (kindIs "invalid" .Values.neo4j.resources.requests) -}}
        {{ $message := printf "Missing neo4j.resources.requests.cpu and neo4j.resources.requests.memory values ! \n %s \n %s" (include "neo4j.resources.minCPUMessage" .) (include "neo4j.resources.minMemoryMessage" .) }}
        {{ fail $message }}
    {{- end -}}

    {{- $cpu := "" }}
    {{- $memory := "" }}
    {{- $isMemorySet := false }}
    {{- $isCPUSet := false }}

    {{- if not (kindIs "invalid" .Values.neo4j.resources.requests) }}

        {{- if not (kindIs "invalid" .Values.neo4j.resources.requests.cpu) }}
            {{- dict "val" .Values.neo4j.resources.requests.cpu "errMsg" "neo4j.resources.requests.cpu cannot be set to \"\"" | include "neo4j.resources.checkForEmptyString" -}}
            {{- $cpu =  .Values.neo4j.resources.requests.cpu | toString }}
            {{- $isCPUSet = true }}
        {{- end -}}

        {{- if not (kindIs "invalid" .Values.neo4j.resources.requests.memory) }}
            {{- dict "val" .Values.neo4j.resources.requests.memory "errMsg" "neo4j.resources.requests.memory cannot be set to \"\"" | include "neo4j.resources.checkForEmptyString" -}}
            {{- $memory =  .Values.neo4j.resources.requests.memory | toString }}
            {{- $isMemorySet = true }}
        {{- end -}}

    {{- end -}}


    {{- if not $isCPUSet }}
        {{- if kindIs "invalid" .Values.neo4j.resources.cpu -}}
            {{- fail (printf "Empty or Missing neo4j.resources.cpu value \n %s" (include "neo4j.resources.minCPUMessage" .)) -}}
        {{- end -}}
        {{- dict "val" .Values.neo4j.resources.cpu "errMsg" "neo4j.resources.cpu cannot be set to \"\"" | include "neo4j.resources.checkForEmptyString" -}}
        {{- $cpu = .Values.neo4j.resources.cpu | toString }}
    {{- end -}}

    {{- if not $isMemorySet }}
         {{- if kindIs "invalid" .Values.neo4j.resources.memory -}}
            {{- fail (printf "Empty or Missing neo4j.resources.memory value \n %s" (include "neo4j.resources.minMemoryMessage" .)) -}}
        {{- end -}}
        {{- dict "val" .Values.neo4j.resources.memory "errMsg" "neo4j.resources.memory cannot be set to \"\"" | include "neo4j.resources.checkForEmptyString" -}}
        {{- $memory = .Values.neo4j.resources.memory | toString }}
    {{- end -}}

    {{- $requests := dict "cpu" $cpu "memory" $memory -}}
    {{- $ignored := set .Values.neo4j.resources "requests" $requests -}}

    {{/* set limits as same as cpu and memory if not provided by the user */}}
    {{- if kindIs "invalid" .Values.neo4j.resources.limits -}}
       {{- $ignored := set .Values.neo4j.resources "limits" $requests -}}
    {{- end -}}

{{- end -}}

{{- define "neo4j.resources.evaluateCPU" -}}

    {{/* check regex here :- https://regex101.com/r/wJsFcO/1 */}}
    {{ $cpuRegex := "(^\\d+)((\\.?[^\\.a-zA-Z])?)([0-9]*m?$)" }}

    {{- $cpu := .Values.neo4j.resources.requests.cpu | toString }}

    {{- if not (regexMatch $cpuRegex $cpu) -}}
        {{ fail (printf "Invalid cpu value %s\n%s" $cpu (include "neo4j.resources.minCPUMessage" .)) }}
    {{- end -}}

    {{- $cpuRegexValue := regexFind $cpuRegex $cpu -}}
    {{- $cpuFloat := 0.0 -}}
    {{/* cpu="123m" , convert millicore cpu to cpu */}}
    {{- if contains "m" $cpuRegexValue -}}
        {{ $cpuFloat = $cpuRegexValue | replace "m" "" | float64 -}}
        {{ $cpuFloat = divf $cpuFloat 1000 -}}
    {{- else -}}
        {{ $cpuFloat = $cpuRegexValue | float64 }}
    {{- end -}}

    {{- if lt $cpuFloat 0.5 }}
        {{ fail (printf "Provided cpu value %s is less than minimum. \n %s" $cpu (include "neo4j.resources.invalidCPUMessage" .) ) }}
    {{- end -}}
{{- end -}}


{{- define "neo4j.resources.evaluateMemory" -}}


    {{/* check regex here :- https://regex101.com/r/68NEQV/1 */}}
    {{ $memoryRegex := "(^\\d+)((\\.?[^\\.a-zA-Z\\s])?)(\\d*)(([EkMGTP]?|[EKMGTP]i?|e[+-]?\\d*[EKMGTP]?)$)" -}}

    {{- $memory := .Values.neo4j.resources.requests.memory | toString }}

    {{- if not (regexMatch $memoryRegex $memory) -}}
        {{ fail (printf "Invalid memory value %s\n%s" $memory (include "neo4j.resources.minMemoryMessage" .)) }}
    {{- end -}}

    {{- $memoryOrig := regexFind $memoryRegex $memory -}}
    {{- $memory = $memoryOrig -}}
    {{- $memoryFloat := 0.0 -}}

    {{- if contains "i" $memory -}}
        {{- $memory = $memory | replace "i" "" -}}
    {{- end -}}

    {{/* Mininum 2Gi or 2Gb, Converting the value type to Gb or Gi */}}

    {{/* 1kilo = 0.000001G */}}
    {{- if or (contains "K" $memory) (contains "k" $memory) -}}
        {{ $memoryFloat = divf ($memory | replace "K" "" | float64) 1000000 -}}

    {{/* 1mega = 0.001G */}}
    {{- else if contains "M" $memory -}}
        {{ $memoryFloat = divf ($memory | replace "M" "" | float64) 1000 -}}

    {{/* giga */}}
    {{- else if contains "G" $memory -}}
        {{ $memoryFloat = $memory | replace "G" "" | float64 -}}

    {{/* 1tera = 1000G */}}
    {{- else if contains "T" $memory -}}
        {{ $memoryFloat =  mulf ($memory | replace "T" "" | float64) 1000 -}}

    {{/* 1peta = 1000000G */}}
    {{- else if contains "P" $memory -}}
        {{ $memoryFloat = mulf ($memory | replace "P" "" | float64) 1000000 -}}

    {{/* 1exa = 1000000000G */}}
    {{- else if contains "E" $memory -}}
        {{ $memoryFloat = mulf ($memory | replace "E" "" | float64) 1000000000 -}}

    {{/* 1Byte = 0.000000001G */}}
    {{- else -}}
        {{ $memoryFloat = divf ($memory | float64) 1000000000 -}}
    {{- end -}}


    {{- if lt $memoryFloat 2.0 }}
        {{ fail (printf "Provided memory value %s is less than minimum. \n %s" $memoryOrig (include "neo4j.resources.invalidMemoryMessage" .)) }}
    {{- end -}}

{{- end -}}


{{- define "neo4j.resources.minCPUMessage" -}}
Please set cpu to be a minimum of 0.5 or 500m via --set neo4j.resources.cpu=0.5 or --set neo4j.resources.cpu=500m
{{- end -}}

{{- define "neo4j.resources.minMemoryMessage" -}}
Please set memory to be a minimum of 2Gi or 2G via --set neo4j.resources.memory=2Gi or --set neo4j.resources.memory=2G
{{- end -}}

{{- define "neo4j.resources.invalidCPUMessage" -}}
cpu value cannot be less than 0.5 or 500m
{{- end -}}

{{- define "neo4j.resources.invalidMemoryMessage" -}}
memory value cannot be less than 2Gb or 2Gi
{{- end -}}

{{/* Takes a dict with "val" and "message" as input*/}}
{{- define "neo4j.resources.checkForEmptyString" -}}
    {{- if eq (len (trim (.val | toString))) 0 }}
        {{ fail (printf .errMsg) }}
    {{- end -}}
{{- end -}}
