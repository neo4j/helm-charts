{{- define "neo4j.reverseProxy.tlsValidation" -}}
    {{- if and $.Values.reverseProxy.ingress.enabled $.Values.reverseProxy.ingress.tls.enabled -}}
        {{- if empty $.Values.reverseProxy.ingress.tls.config -}}
            {{- if empty $.Values.reverseProxy.ingress.annotations -}}
                {{ fail (printf "When TLS is enabled, either tls.config or ingress annotations must be provided") }}
            {{- end -}}
        {{- else -}}
            {{- range $.Values.reverseProxy.ingress.tls.config -}}
                {{- $value := . -}}
                {{- if and (not (kindIs "invalid" $value.secretName)) (empty ($value.secretName | trim)) -}}
                    {{ fail (printf "Empty secretName for tls config") }}
                {{- end -}}
            {{- end -}}
        {{- end -}}
    {{- end -}}
{{- end -}}
