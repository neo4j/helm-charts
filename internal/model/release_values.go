package model

type HelmValues struct {
	FullnameOverride         string              `yaml:"fullnameOverride,omitempty"`
	NameOverride             string              `yaml:"nameOverride,omitempty"`
	Neo4J                    Neo4J               `yaml:"neo4j,omitempty"`
	Volumes                  Volumes             `yaml:"volumes,omitempty"`
	AdditionalVolumes        []interface{}       `yaml:"additionalVolumes,omitempty"`
	AdditionalVolumeMounts   []interface{}       `yaml:"additionalVolumeMounts,omitempty"`
	NodeSelector             map[string]string   `yaml:"nodeSelector,omitempty"`
	DisableLookups           bool                `default:"false" yaml:"disableLookups"`
	Services                 Services            `yaml:"services,omitempty"`
	Config                   map[string]string   `yaml:"config,omitempty"`
	SecurityContext          SecurityContext     `yaml:"securityContext,omitempty"`
	ContainerSecurityContext SecurityContext     `yaml:"containerSecurityContext,omitempty"`
	ReadinessProbe           ReadinessProbe      `yaml:"readinessProbe,omitempty"`
	LivenessProbe            LivenessProbe       `yaml:"livenessProbe,omitempty"`
	StartupProbe             StartupProbe        `yaml:"startupProbe,omitempty"`
	Ssl                      Ssl                 `yaml:"ssl,omitempty"`
	ClusterDomain            string              `yaml:"clusterDomain,omitempty"`
	Image                    Image               `yaml:"image,omitempty"`
	Statefulset              Statefulset         `yaml:"statefulset,omitempty"`
	Env                      Env                 `yaml:"env,omitempty"`
	PodSpec                  PodSpec             `yaml:"podSpec,omitempty"`
	LogInitialPassword       bool                `yaml:"logInitialPassword,omitempty"`
	Jvm                      Jvm                 `yaml:"jvm,omitempty"`
	Logs                     Logging             `yaml:"logs,omitempty"`
	LdapPasswordFromSecret   string              `yaml:"ldapPasswordFromSecret,omitempty"`
	LdapPasswordMountPath    string              `yaml:"ldapPasswordMountPath,omitempty"`
	ApocCredentials          ApocCredentials     `yaml:"apoc_credentials,omitempty"`
	PodDisruptionBudget      PodDisruptionBudget `yaml:"podDisruptionBudget"`
	ServiceMonitor           ServiceMonitor      `yaml:"serviceMonitor"`
	Analytics                Analytics           `yaml:"analytics"`
}

type Operations struct {
	EnableServer bool              `yaml:"enableServer"`
	Image        string            `yaml:"image"`
	Protocol     string            `yaml:"protocol"`
	Labels       map[string]string `yaml:"labels"`
}

type Analytics struct {
	Enabled bool `yaml:"enabled"`
	Type    Type `yaml:"type"`
}

type Type struct {
	Name string `yaml:"name"`
}

type ServiceMonitor struct {
	Enabled           bool                   `yaml:"enabled"`
	Labels            map[string]string      `yaml:"labels,omitempty"`
	JobLabel          string                 `yaml:"jobLabel,omitempty"`
	Interval          string                 `yaml:"interval,omitempty"`
	Port              string                 `yaml:"port,omitempty"`
	Path              string                 `yaml:"path,omitempty"`
	NamespaceSelector NamespaceSelector      `yaml:"namespaceSelector"`
	TargetLabels      []string               `yaml:"targetLabels,omitempty"`
	Selector          ServiceMonitorSelector `yaml:"selector,omitempty"`
}

type ServiceMonitorSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels,omitempty"`
}
type NamespaceSelector struct {
	Any        bool     `yaml:"any"`
	MatchNames []string `yaml:"matchNames,omitempty"`
}

type PodDisruptionBudget struct {
	Enabled          bool               `yaml:"enabled"`
	MatchLabels      map[string]string  `yaml:"matchLabels,omitempty"`
	MatchExpressions []MatchExpressions `yaml:"matchExpressions,omitempty"`
	Labels           map[string]string  `yaml:"labels,omitempty"`
	MinAvailable     string             `yaml:"minAvailable,omitempty"`
	MaxUnavailable   string             `yaml:"maxUnavailable,omitempty"`
}
type PodSpec struct {
	Annotations                   map[string]string          `yaml:"annotations,omitempty"`
	NodeAffinity                  map[string]interface{}     `yaml:"nodeAffinity,omitempty"`
	PodAntiAffinity               interface{}                `yaml:"podAntiAffinity,omitempty"`
	Tolerations                   []interface{}              `yaml:"tolerations,omitempty"`
	PriorityClassName             string                     `yaml:"priorityClassName,omitempty"`
	DNSPolicy                     string                     `yaml:"dnsPolicy,omitempty"`
	Loadbalancer                  string                     `yaml:"loadbalancer,omitempty"`
	ServiceAccountName            string                     `yaml:"serviceAccountName,omitempty"`
	TerminationGracePeriodSeconds int                        `yaml:"terminationGracePeriodSeconds,omitempty"`
	InitContainers                []interface{}              `yaml:"initContainers,omitempty"`
	Containers                    []interface{}              `yaml:"containers,omitempty"`
	TopologySpreadConstraints     []TopologySpreadConstraint `yaml:"topologySpreadConstraints,omitempty"`
}

type TopologySpreadConstraint struct {
	MaxSkew           int    `yaml:"maxSkew"`
	TopologyKey       string `yaml:"topologyKey"`
	WhenUnsatisfiable string `yaml:"whenUnsatisfiable"`
}

type ApocCredentials struct {
	Jdbc          map[string]string `yaml:"jdbc,omitempty"`
	Elasticsearch map[string]string `yaml:"elasticsearch,omitempty"`
}

type Resources struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}
type Neo4J struct {
	Name                          string      `yaml:"name,omitempty"`
	Password                      string      `yaml:"password,omitempty"`
	PasswordFromSecret            string      `yaml:"passwordFromSecret,omitempty"`
	Edition                       string      `yaml:"edition,omitempty"`
	MinimumClusterSize            int         `yaml:"minimumClusterSize,omitempty"`
	AcceptLicenseAgreement        string      `yaml:"acceptLicenseAgreement,omitempty"`
	OfflineMaintenanceModeEnabled bool        `yaml:"offlineMaintenanceModeEnabled,omitempty"`
	Resources                     Resources   `yaml:"resources,omitempty"`
	Labels                        interface{} `yaml:"labels,omitempty"`
	Operations                    Operations  `yaml:"operations,omitempty"`
}
type Requests struct {
	Storage string `yaml:"storage,omitempty"`
}
type MatchLabels struct {
	App                    string `yaml:"app,omitempty"`
	HelmNeo4JComVolumeRole string `yaml:"helm.neo4j.com/volume-role,omitempty"`
}
type SelectorTemplate struct {
	MatchLabels MatchLabels `yaml:"matchLabels,omitempty"`
}
type Selector struct {
	StorageClassName string           `yaml:"storageClassName,omitempty"`
	AccessModes      []string         `yaml:"accessModes,omitempty"`
	Requests         Requests         `yaml:"requests,omitempty"`
	SelectorTemplate SelectorTemplate `yaml:"selectorTemplate,omitempty"`
}
type DefaultStorageClass struct {
	AccessModes []string `yaml:"accessModes,omitempty"`
	Requests    Requests `yaml:"requests,omitempty"`
}
type Dynamic struct {
	StorageClassName string   `yaml:"storageClassName,omitempty"`
	AccessModes      []string `yaml:"accessModes,omitempty"`
	Requests         Requests `yaml:"requests,omitempty"`
}
type Volume struct {
	SetOwnerAndGroupWritableFilePermissions bool `yaml:"setOwnerAndGroupWritableFilePermissions,omitempty"`
}
type VolumeClaimTemplate struct {
}
type Data struct {
	Mode                string              `yaml:"mode,omitempty"`
	Selector            Selector            `yaml:"selector,omitempty"`
	DefaultStorageClass DefaultStorageClass `yaml:"defaultStorageClass,omitempty"`
	Dynamic             Dynamic             `yaml:"dynamic,omitempty"`
	Volume              Volume              `yaml:"volume,omitempty"`
	VolumeClaimTemplate VolumeClaimTemplate `yaml:"volumeClaimTemplate,omitempty"`
	DisableSubPath      bool                `yaml:"disableSubPathExpr,omitempty"`
	Labels              map[string]string   `yaml:"labels,omitempty"`
}
type Share struct {
	Name string `yaml:"name,omitempty"`
}
type Backups struct {
	Mode           string            `yaml:"mode,omitempty"`
	Share          Share             `yaml:"share,omitempty"`
	DisableSubPath bool              `yaml:"disableSubPathExpr,omitempty"`
	Labels         map[string]string `yaml:"labels,omitempty"`
}
type Logs struct {
	Mode           string            `yaml:"mode,omitempty"`
	Share          Share             `yaml:"share,omitempty"`
	DisableSubPath bool              `yaml:"disableSubPathExpr,omitempty"`
	Labels         map[string]string `yaml:"labels,omitempty"`
}
type Metrics struct {
	Mode           string            `yaml:"mode,omitempty"`
	Share          Share             `yaml:"share,omitempty"`
	DisableSubPath bool              `yaml:"disableSubPathExpr,omitempty"`
	Labels         map[string]string `yaml:"labels,omitempty"`
}
type Import struct {
	Mode           string            `yaml:"mode,omitempty"`
	Share          Share             `yaml:"share,omitempty"`
	DisableSubPath bool              `yaml:"disableSubPathExpr,omitempty"`
	Labels         map[string]string `yaml:"labels,omitempty"`
}
type Licenses struct {
	Mode           string            `yaml:"mode,omitempty"`
	Share          Share             `yaml:"share,omitempty"`
	DisableSubPath bool              `yaml:"disableSubPathExpr,omitempty"`
	Labels         map[string]string `yaml:"labels,omitempty"`
}
type Volumes struct {
	Data     Data     `yaml:"data,omitempty"`
	Backups  Backups  `yaml:"backups,omitempty"`
	Logs     Logs     `yaml:"logs,omitempty"`
	Metrics  Metrics  `yaml:"metrics,omitempty"`
	Import   Import   `yaml:"import,omitempty"`
	Licenses Licenses `yaml:"licenses,omitempty"`
}
type Annotations struct {
}
type Default struct {
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
type Spec struct {
	Type string `yaml:"type,omitempty"`
}
type Port struct {
	Enabled    bool   `yaml:"enabled"`
	Port       int    `yaml:"port"`
	TargetPort int    `yaml:"targetPort"`
	NodePort   int    `yaml:"nodePort"`
	Name       string `yaml:"name"`
}

type Ports struct {
	HTTP   Port `yaml:"http"`
	HTTPS  Port `yaml:"https"`
	Bolt   Port `yaml:"bolt,omitempty"`
	Backup Port `yaml:"backup,omitempty"`
}
type ServiceSelector struct {
	HelmNeo4JComNeo4JLoadbalancer string `yaml:"helm.neo4j.com/neo4j.loadbalancer,omitempty"`
}
type Neo4jService struct {
	Enabled      bool              `yaml:"enabled,omitempty" default:"true"`
	Annotations  map[string]string `yaml:"annotations,omitempty"`
	Spec         Spec              `yaml:"spec,omitempty"`
	Ports        Ports             `yaml:"ports"`
	Selector     ServiceSelector   `yaml:"selector,omitempty"`
	MultiCluster bool              `yaml:"multiCluster,omitempty"`
}
type Admin struct {
	Enabled     bool              `yaml:"enabled,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
	Spec        Spec              `yaml:"spec,omitempty"`
}
type Internals struct {
	Enabled     bool              `yaml:"enabled,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
type Services struct {
	Default   Default      `yaml:"default,omitempty"`
	Neo4j     Neo4jService `yaml:"neo4j,omitempty"`
	Admin     Admin        `yaml:"admin,omitempty"`
	Internals Internals    `yaml:"internals,omitempty"`
}

type SecurityContext struct {
	RunAsNonRoot        bool         `yaml:"runAsNonRoot,omitempty"`
	RunAsUser           int          `yaml:"runAsUser,omitempty"`
	RunAsGroup          int          `yaml:"runAsGroup,omitempty"`
	FsGroup             int          `yaml:"fsGroup,omitempty"`
	FsGroupChangePolicy string       `yaml:"fsGroupChangePolicy,omitempty"`
	Capabilities        Capabilities `yaml:"capabilities,omitempty"`
}

type Capabilities struct {
	Drop []string `yaml:"drop,omitempty"`
}

type ReadinessProbe struct {
	FailureThreshold int `yaml:"failureThreshold,omitempty"`
	TimeoutSeconds   int `yaml:"timeoutSeconds,omitempty"`
	PeriodSeconds    int `yaml:"periodSeconds,omitempty"`
}
type LivenessProbe struct {
	FailureThreshold int `yaml:"failureThreshold,omitempty"`
	TimeoutSeconds   int `yaml:"timeoutSeconds,omitempty"`
	PeriodSeconds    int `yaml:"periodSeconds,omitempty"`
}
type StartupProbe struct {
	FailureThreshold int `yaml:"failureThreshold,omitempty"`
	PeriodSeconds    int `yaml:"periodSeconds,omitempty"`
}
type PrivateKey struct {
	SecretName interface{} `yaml:"secretName,omitempty"`
	SubPath    interface{} `yaml:"subPath,omitempty"`
}
type PublicCertificate struct {
	SecretName interface{} `yaml:"secretName,omitempty"`
	SubPath    interface{} `yaml:"subPath,omitempty"`
}
type TrustedCerts struct {
	Sources []interface{} `yaml:"sources,omitempty"`
}
type RevokedCerts struct {
	Sources []interface{} `yaml:"sources,omitempty"`
}
type Bolt struct {
	PrivateKey        PrivateKey        `yaml:"privateKey,omitempty"`
	PublicCertificate PublicCertificate `yaml:"publicCertificate,omitempty"`
	TrustedCerts      TrustedCerts      `yaml:"trustedCerts,omitempty"`
	RevokedCerts      RevokedCerts      `yaml:"revokedCerts,omitempty"`
}
type HTTPS struct {
	PrivateKey        PrivateKey        `yaml:"privateKey,omitempty"`
	PublicCertificate PublicCertificate `yaml:"publicCertificate,omitempty"`
	TrustedCerts      TrustedCerts      `yaml:"trustedCerts,omitempty"`
	RevokedCerts      RevokedCerts      `yaml:"revokedCerts,omitempty"`
}
type Ssl struct {
	Bolt  Bolt  `yaml:"bolt,omitempty"`
	HTTPS HTTPS `yaml:"https,omitempty"`
}
type Image struct {
	ImagePullPolicy  string   `yaml:"imagePullPolicy,omitempty"`
	CustomImage      string   `yaml:"customImage,omitempty"`
	ImagePullSecrets []string `yaml:"imagePullSecrets,omitempty"`
}
type Metadata struct {
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
type Statefulset struct {
	Metadata Metadata `yaml:"metadata,omitempty"`
}
type Env struct {
}

type Jvm struct {
	UseNeo4JDefaultJvmArguments bool     `yaml:"useNeo4jDefaultJvmArguments,omitempty"`
	AdditionalJvmArguments      []string `yaml:"additionalJvmArguments,omitempty"`
}
type Logging struct {
	ServerLogsXML string `yaml:"serverLogsXml,omitempty"`
	UserLogsXML   string `yaml:"userLogsXml,omitempty"`
}

type PodAntiAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution []RequiredDuringSchedulingIgnoredDuringExecution `yaml:"requiredDuringSchedulingIgnoredDuringExecution"`
}
