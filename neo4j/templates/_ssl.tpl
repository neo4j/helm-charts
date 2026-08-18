{{- define "neo4j.ssl.volumesFromSecrets" -}}
{{- range $name, $sslSpec := . -}}
{{- if ( or $sslSpec.privateKey.secretName $sslSpec.publicCertificate.secretName ) }}
- name: "{{ $name }}-certificates"
  projected:
    sources:
      - secret:
          name: "{{ required "When ssl.{{ $name }}.privateKey is set then ssl.{{ $name }}.publicCertificate.secretName must also be provided" $sslSpec.publicCertificate.secretName }}"
          items:
            - key: "{{ $sslSpec.publicCertificate.subPath | default "public.crt" }}"
              path: public.crt
      - secret:
          name: "{{ required "When ssl.{{ $name }}.publicCertificate is set then ssl.{{ $name }}.privateKey.secretName must also be provided" $sslSpec.privateKey.secretName }}"
          items:
            - key: "{{ $sslSpec.privateKey.subPath | default "private.key" }}"
              path: private.key
{{- if $sslSpec.trustedCerts.sources }}
- name: "{{ $name }}-trusted"
  projected:
    defaultMode: 0440
    {{ $sslSpec.trustedCerts | toYaml | nindent 4 }}
{{- end }}
{{- if $sslSpec.revokedCerts.sources -}}
- name: "{{ $name }}-revoked"
  projected:
    defaultMode: 0440
    {{ $sslSpec.revokedCerts | toYaml | nindent 4 }}
{{/* blank line, important! */}}{{ end -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "neo4j.ssl.volumeMountsFromSecrets" -}}
{{- range $name, $sslSpec := . -}}
{{- if ( or $sslSpec.privateKey.secretName $sslSpec.publicCertificate.secretName ) }}
- name: "{{ $name }}-certificates"
  mountPath: "/var/lib/neo4j/certificates/{{ $name }}"
  readOnly: true
{{- if $sslSpec.trustedCerts.sources }}
- name: "{{ $name }}-trusted"
  mountPath: "/var/lib/neo4j/certificates/{{ $name }}/trusted"
  readOnly: true
{{- end -}}
{{- if $sslSpec.revokedCerts.sources }}
- name: "{{ $name }}-revoked"
  mountPath: "/var/lib/neo4j/certificates/{{ $name }}/revoked"
  readOnly: true
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
