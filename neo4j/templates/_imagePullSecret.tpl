{{/* To see imagePullSecret registrys in the statefulset*/}}
{{- define "neo4j.imagePullSecrets" -}}
     {{/* add imagePullSecrets only when the list of imagePullSecrets is not emtpy and does not have all the entries as empty */}}
    {{- if and (not (empty .)) (ne (len ( . | join "" | trim)) 0) }}
imagePullSecrets:
    {{- range $name := . -}}
        {{/* do not allow empty imagePullSecret registries */}}
        {{- if ne (len ($name | trim)) 0 }}
 - name: "{{ $name }}"
        {{- end -}}
    {{- end -}}
    {{- end -}}
{{- end -}}

{{/* Generates the .dockerconfigjson value for the respective .Values.neo4j.image.imageCredentials */}}
{{- define "neo4j.imagePullSecret.dockerConfigJson" }}
{{- printf "{\"auths\":{\"%s\":{\"username\":\"%s\",\"password\":\"%s\",\"email\":\"%s\",\"auth\":\"%s\"}}}" .registry .username .password .email (printf "%s:%s" .username .password | b64enc) | b64enc }}
{{- end }}

{{/* checks and throws error if registry ,username , email and name imageCredentials fields are empty */}}
{{- define "neo4j.imageCredentials.checkForEmptyFields" -}}

    {{- if .Values.image.imageCredentials -}}
        {{- $errorList := list -}}
        {{- range $index, $element := .Values.image.imageCredentials }}

            {{- if  empty ($element.registry | trim) -}}
                {{- $errorList = append $errorList (printf "Registry field cannot be empty for imageCredential \"%s\"" $element.name) -}}
            {{- end -}}

            {{- if  empty ($element.username | trim) -}}
                {{- $errorList = append $errorList (printf "Username field cannot be empty for imageCredential \"%s\"" $element.name) -}}
            {{- end -}}

            {{- if  empty ($element.password | trim) -}}
                {{- $errorList = append $errorList (printf "Password field cannot be empty for imageCredential \"%s\"" $element.name) -}}
            {{- end -}}

            {{- if  empty ($element.email | trim) -}}
                {{- $errorList = append $errorList (printf "Email field cannot be empty for imageCredential \"%s\"" $element.name) -}}
            {{- end -}}

            {{- if  empty ($element.name | trim) -}}
                {{- $errorList = append $errorList "name field cannot be empty for an imageCredential" -}}
            {{- end -}}

        {{- end -}}

        {{- if not (empty $errorList) -}}
           {{ fail (printf "%s" ($errorList | join "\n")) }}
        {{- end -}}

    {{- end -}}
{{- end -}}

{{/* checkForDuplicates throws an error if there are duplicate 'names' found in imageCredential list */}}
{{- define "neo4j.imageCredentials.checkForDuplicates" -}}

    {{- if .Values.image.imageCredentials -}}

        {{- $nameList := list -}}

        {{- range $index, $element := .Values.image.imageCredentials }}
            {{- $nameList = append $nameList $element.name -}}
        {{- end -}}

        {{- $nameList = $nameList | uniq -}}

        {{- if ne (len $nameList) (len .Values.image.imageCredentials) -}}
            {{ fail (printf "Duplicate \"names\" found in imageCredentials list. Please remove duplicates") }}
        {{- end -}}

    {{- end -}}
{{- end -}}

{{/* getImageCredential returns an imageCredential for the given name */}}
{{- define "neo4j.imageCredentials.getImageCredential" -}}

    {{- $imagePullSecretName := .imagePullSecret -}}
    {{- range $index, $element := .imageCredentials -}}
        {{- if eq $element.name $imagePullSecretName -}}
            {{- $element | toYaml -}}
        {{- end -}}
    {{- end -}}
{{- end -}}
