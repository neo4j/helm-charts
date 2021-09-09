{{/* vim: set filetype=mustache: */}}
{{/*
Convert a neo4j.conf properties text into valid yaml
*/}}
{{- define "neo4j.configYaml" -}}
  {{- regexReplaceAll "(?m)^([^=]*?)=" ( regexReplaceAllLiteral "\\s*(#|dbms\\.jvm\\.additional).*" . "" )  "${1}: " | trim | replace ": true\n" ": 'true'\n" | replace ": false\n" ": 'false'\n"  |  replace ": yes\n" ": 'yes'\n" |  replace ": no\n" ": 'no'\n" }}
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

{{- define "neo4j.image" -}}
{{- $isEnterprise := required "neo4j.edition must be specified" .Values.neo4j.edition | regexMatch "(?i)enterprise" -}}
{{- template "neo4j.checkLicenseAgreement" . -}}
{{- if .Values.image.customImage }}{{ .Values.image.customImage }}{{ else }}neo4j:{{ .Chart.AppVersion }}{{ if $isEnterprise }}-enterprise{{ end }}{{ end -}}
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
