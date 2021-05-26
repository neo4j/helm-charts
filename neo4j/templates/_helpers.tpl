{{/* vim: set filetype=mustache: */}}
{{/*
Convert a neo4j.conf properties text into valid yaml
*/}}
{{- define "neo4j.configYaml" -}}
  {{- regexReplaceAll "(?m)^([^=]*?)=" ( regexReplaceAllLiteral "\\s*(#|dbms\\.jvm\\.additional).*" . "" )  "${1}: " | trim | replace ": true\n" ": 'true'\n" | replace ": false\n" ": 'false'\n"  |  replace ": yes\n" ": 'yes'\n" |  replace ": no\n" ": 'no'\n" }}
{{- end -}}

{{- define "neo4j.configJvmAdditionalYaml" -}}
  {{- /* This collects together all dbms.jvm.additional entries */}}
dbms.jvm.additional: |- {{- range ( regexFindAll "(?m)^\\s*(dbms\\.jvm\\.additional=).+" . -1 ) }}{{ trim . | replace "dbms.jvm.additional=" "" | trim | nindent 2 }}{{- end }}
{{- end -}}

{{- define "neo4j.appName" -}}
  {{- .Values.neo4j.name | default .Release.Name }}
{{- end -}}

{{/*
If no password is set in `Values.neo4j.password` generates a new random password and modifies Values.neo4j so that the same password is available everywhere
*/}}
{{- define "neo4j.password" -}}
  {{- if not .Values.neo4j.password }}
    {{- $password :=  randAlphaNum 14 }}
    {{- $ignored := set .Values.neo4j "password" $password }}
  {{- end -}}
  {{- .Values.neo4j.password }}
{{- end -}}
