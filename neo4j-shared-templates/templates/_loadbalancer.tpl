{{- define "neo4j.services.neo4j" -}}
{{- if .Values.enabled }}
# Service for applications that need access to neo4j
apiVersion: v1
kind: Service
metadata:
  name: "{{ .Release.Name }}-neo4j"
  namespace: "{{ .Release.Namespace }}"
  labels:
    helm.neo4j.com/neo4j.name: "{{ template "neo4j.name" $ }}"
    app: "{{ template "neo4j.appName" . }}"
    helm.neo4j.com/service: "neo4j"
  {{- with .Values.annotations }}
  annotations: {{ toYaml . | nindent 4 }}
  {{- end }}
spec:
  type: "{{ .Values.type }}"
  {{- with .Values.clusterIP }}
  clusterIP: "{{ . }}"
  {{- end }}
  {{- with .Values.loadBalancerIP }}
  loadBalancerIP: "{{ . }}"
  {{- end }}
  sessionAffinity: None
  externalTrafficPolicy: Local
  ports:
    {{- with .Values.ports }}
    {{- if .http.enabled }}
    - protocol: TCP
      port: 7474
      targetPort: 7474
      name: http
    {{- end }}
    {{- if .https.enabled }}
    - protocol: TCP
      port: 7473
      targetPort: 7473
      name: https
    {{- end }}
    {{- if .bolt.enabled }}
    - protocol: TCP
      port: 7687
      targetPort: 7687
      name: tcp-bolt
    {{- end }}
    {{- end }}
  selector:
    app: "{{ template "neo4j.appName" . | required "neo4j.name must be specified" }}"
    {{- with .Values.selector }}
    {{- . | toYaml | nindent 4 }}
    {{- end }}
{{- end -}}
{{- end -}}
