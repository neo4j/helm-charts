package model

type Neo4jReverseProxyValues struct {
	NameOverride     string       `yaml:"nameOverride,omitempty"`
	FullnameOverride string       `yaml:"fullnameOverride,omitempty"`
	ReverseProxy     ReverseProxy `yaml:"reverseProxy,omitempty"`
}

type ReverseProxy struct {
	Image            string                `yaml:"image,omitempty"`
	ImagePullSecrets []string              `yaml:"imagePullSecrets,omitempty"`
	ServiceName      string                `yaml:"serviceName,omitempty"`
	Namespace        string                `yaml:"namespace,omitempty"`
	Domain           string                `yaml:"domain,omitempty"`
	Ingress          Ingress               `yaml:"ingress,omitempty"`
	PodLabels        map[string]string     `yaml:"podLabels,omitempty"`
	NodeSelector     map[string]string     `yaml:"nodeSelector,omitempty"`
	Resources        ReverseProxyResources `yaml:"resources,omitempty"`
}

type ReverseProxyResources struct {
	Requests ReverseProxyRequests `yaml:"requests,omitempty"`
	Limits   ReverseProxyLimits   `yaml:"limits,omitempty"`
}

type ReverseProxyRequests struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

type ReverseProxyLimits struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

type Ingress struct {
	Enabled     bool              `yaml:"enabled"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
	TLS         TLS               `yaml:"tls,omitempty"`
	Host        string            `yaml:"host,omitempty"`
}

type TLS struct {
	Enabled bool     `yaml:"enabled"`
	Config  []Config `yaml:"config,omitempty"`
}

type Config struct {
	Hosts      []string `yaml:"hosts,omitempty"`
	SecretName string   `yaml:"secretName,omitempty"`
}
