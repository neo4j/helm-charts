{{- define "neo4j.reverseProxy.tlsValidation" -}}
    {{- if and $.Values.reverseProxy.ingress.enabled $.Values.reverseProxy.ingress.tls.enabled -}}
        {{- if empty $.Values.reverseProxy.ingress.tls.config -}}
            {{ fail (printf "Empty tls config !!") }}
        {{- end -}}
        {{- range $.Values.reverseProxy.ingress.tls.config -}}
            {{- $value := . -}}
            {{- if kindIs "invalid" $value.secretName  -}}
                {{ fail (printf "Missing secretName for tls config") }}
            {{- end -}}
            {{- if empty ($value.secretName | trim)  -}}
                {{ fail (printf "Empty secretName for tls config") }}
            {{- end -}}
        {{- end -}}
    {{- end -}}
{{- end -}}
