package model

type Neo4jReverseProxyValues struct {
	NameOverride     string       `yaml:"nameOverride,omitempty"`
	FullnameOverride string       `yaml:"fullnameOverride,omitempty"`
	ReverseProxy     ReverseProxy `yaml:"reverseProxy,omitempty"`
}

type ReverseProxy struct {
	Image       string  `yaml:"image,omitempty"`
	ServiceName string  `yaml:"serviceName,omitempty"`
	Namespace   string  `yaml:"namespace,omitempty"`
	Domain      string  `yaml:"domain,omitempty"`
	Ingress     Ingress `yaml:"ingress,omitempty"`
}

type Ingress struct {
	Enabled     bool              `yaml:"enabled"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
	TLS         TLS               `yaml:"tls,omitempty"`
}

type TLS struct {
	Enabled bool     `yaml:"enabled"`
	Config  []Config `yaml:"config,omitempty"`
}

type Config struct {
	Hosts      []string `yaml:"hosts,omitempty"`
	SecretName string   `yaml:"secretName,omitempty"`
}
