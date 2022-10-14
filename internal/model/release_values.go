package model

import v1 "k8s.io/api/core/v1"

type HelmValues struct {
	Neo4J                  Neo4J             `yaml:"neo4j,omitempty"`
	Volumes                Volumes           `yaml:"volumes,omitempty"`
	AdditionalVolumes      []interface{}     `yaml:"additionalVolumes,omitempty"`
	AdditionalVolumeMounts []interface{}     `yaml:"additionalVolumeMounts,omitempty"`
	NodeSelector           v1.NodeSelector   `yaml:"nodeSelector,omitempty"`
	Services               Services          `yaml:"services,omitempty"`
	Config                 map[string]string `yaml:"config,omitempty"`
	SecurityContext        SecurityContext   `yaml:"securityContext,omitempty"`
	ReadinessProbe         ReadinessProbe    `yaml:"readinessProbe,omitempty"`
	LivenessProbe          LivenessProbe     `yaml:"livenessProbe,omitempty"`
	StartupProbe           StartupProbe      `yaml:"startupProbe,omitempty"`
	Ssl                    Ssl               `yaml:"ssl,omitempty"`
	ClusterDomain          string            `yaml:"clusterDomain,omitempty"`
	Image                  Image             `yaml:"image,omitempty"`
	Statefulset            Statefulset       `yaml:"statefulset,omitempty"`
	Env                    Env               `yaml:"env,omitempty"`
	PodSpec                v1.PodSpec        `yaml:"podSpec,omitempty"`
	LogInitialPassword     bool              `yaml:"logInitialPassword,omitempty"`
	Jvm                    Jvm               `yaml:"jvm,omitempty"`
	Logs                   Logging           `yaml:"logs,omitempty"`
}
type Resources struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}
type Neo4J struct {
	Name                          string      `yaml:"name,omitempty"`
	Password                      string      `yaml:"password,omitempty"`
	Edition                       string      `yaml:"edition,omitempty"`
	MinimumClusterSize            int         `yaml:"minimumClusterSize,omitempty"`
	AcceptLicenseAgreement        string      `yaml:"acceptLicenseAgreement,omitempty"`
	OfflineMaintenanceModeEnabled bool        `yaml:"offlineMaintenanceModeEnabled,omitempty"`
	Resources                     Resources   `yaml:"resources,omitempty"`
	Labels                        interface{} `yaml:"labels,omitempty"`
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
}
type Share struct {
	Name string `yaml:"name,omitempty"`
}
type Backups struct {
	Mode  string `yaml:"mode,omitempty"`
	Share Share  `yaml:"share,omitempty"`
}
type Logs struct {
	Mode  string `yaml:"mode,omitempty"`
	Share Share  `yaml:"share,omitempty"`
}
type Metrics struct {
	Mode  string `yaml:"mode,omitempty"`
	Share Share  `yaml:"share,omitempty"`
}
type Import struct {
	Mode  string `yaml:"mode,omitempty"`
	Share Share  `yaml:"share,omitempty"`
}
type Licenses struct {
	Mode  string `yaml:"mode,omitempty"`
	Share Share  `yaml:"share,omitempty"`
}
type Volumes struct {
	Data     Data     `yaml:"data,omitempty"`
	Backups  Backups  `yaml:"backups,omitempty"`
	Logs     Logs     `yaml:"logs,omitempty"`
	Metrics  Metrics  `yaml:"metrics,omitempty"`
	Import   Import   `yaml:"import,omitempty"`
	Licenses Licenses `yaml:"licenses,omitempty"`
}
type NodeSelector struct {
}
type Annotations struct {
}
type Default struct {
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
type Spec struct {
	Type string `yaml:"type,omitempty"`
}
type HTTPPort struct {
	Enabled bool `yaml:"enabled,omitempty"`
}
type HTTPSPort struct {
	Enabled bool `yaml:"enabled,omitempty"`
}
type BoltPort struct {
	Enabled bool `yaml:"enabled,omitempty"`
}
type BackupPort struct {
	Enabled bool `yaml:"enabled,omitempty"`
}
type Ports struct {
	HTTP   HTTPPort   `yaml:"http,omitempty"`
	HTTPS  HTTPSPort  `yaml:"https,omitempty"`
	Bolt   BoltPort   `yaml:"bolt,omitempty"`
	Backup BackupPort `yaml:"backup,omitempty"`
}
type ServiceSelector struct {
	HelmNeo4JComNeo4JLoadbalancer string `yaml:"helm.neo4j.com/neo4j.loadbalancer,omitempty"`
}
type Neo4JService struct {
	Enabled      bool              `yaml:"enabled,omitempty"`
	Annotations  map[string]string `yaml:"annotations,omitempty"`
	Spec         Spec              `yaml:"spec,omitempty"`
	Ports        Ports             `yaml:"ports,omitempty"`
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
	Default   Default   `yaml:"default,omitempty"`
	Neo4J     Neo4J     `yaml:"neo4j,omitempty"`
	Admin     Admin     `yaml:"admin,omitempty"`
	Internals Internals `yaml:"internals,omitempty"`
}

type SecurityContext struct {
	RunAsNonRoot        bool   `yaml:"runAsNonRoot,omitempty"`
	RunAsUser           int    `yaml:"runAsUser,omitempty"`
	RunAsGroup          int    `yaml:"runAsGroup,omitempty"`
	FsGroup             int    `yaml:"fsGroup,omitempty"`
	FsGroupChangePolicy string `yaml:"fsGroupChangePolicy,omitempty"`
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
	ImagePullPolicy string `yaml:"imagePullPolicy,omitempty"`
	CustomImage     string `yaml:"customImage,omitempty"`
}
type Metadata struct {
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
type Statefulset struct {
	Metadata Metadata `yaml:"metadata,omitempty"`
}
type Env struct {
}

type PodSpec struct {
	Annotations                   map[string]string `yaml:"annotations,omitempty"`
	NodeAffinity                  v1.NodeAffinity   `yaml:"nodeAffinity,omitempty"`
	PodAntiAffinity               bool              `yaml:"podAntiAffinity,omitempty"`
	Tolerations                   []v1.Toleration   `yaml:"tolerations,omitempty"`
	PriorityClassName             string            `yaml:"priorityClassName,omitempty"`
	Loadbalancer                  string            `yaml:"loadbalancer,omitempty"`
	ServiceAccountName            string            `yaml:"serviceAccountName,omitempty"`
	TerminationGracePeriodSeconds int               `yaml:"terminationGracePeriodSeconds,omitempty"`
	InitContainers                []v1.Container    `yaml:"initContainers,omitempty"`
	Containers                    []v1.Container    `yaml:"containers,omitempty"`
}
type Jvm struct {
	UseNeo4JDefaultJvmArguments bool     `yaml:"useNeo4jDefaultJvmArguments,omitempty"`
	AdditionalJvmArguments      []string `yaml:"additionalJvmArguments,omitempty"`
}
type Logging struct {
	ServerLogsXML string `yaml:"serverLogsXml,omitempty"`
	UserLogsXML   string `yaml:"userLogsXml,omitempty"`
}
