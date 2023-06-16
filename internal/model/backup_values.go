package model

type Neo4jBackupValues struct {
	NameOverride       string                 `yaml:"nameOverride,omitempty"`
	FullnameOverride   string                 `yaml:"fullnameOverride,omitempty"`
	DisableLookups     bool                   `yaml:"disableLookups" default:"false"`
	Neo4J              Neo4jBackupNeo4j       `yaml:"neo4j"`
	Backup             Backup                 `yaml:"backup"`
	ServiceAccountName string                 `yaml:"serviceAccountName"`
	TempVolume         map[string]interface{} `yaml:"tempVolume"`
	SecurityContext    SecurityContext        `yaml:"securityContext"`
}

type Neo4jBackupNeo4j struct {
	Image                      string            `yaml:"image" default:"neo4jbuildservice/helm-charts"`
	ImageTag                   string            `yaml:"imageTag" default:"backup"`
	PodLabels                  map[string]string `yaml:"podLabels,omitempty"`
	PodAnnotations             map[string]string `yaml:"podAnnotations,omitempty"`
	JobSchedule                string            `yaml:"jobSchedule" default:"* * * * *"`
	SuccessfulJobsHistoryLimit int               `yaml:"successfulJobsHistoryLimit" default:"3"`
	FailedJobsHistoryLimit     int               `yaml:"failedJobsHistoryLimit" default:"1"`
	BackoffLimit               int               `yaml:"backoffLimit" default:"6"`
	Labels                     map[string]string `yaml:"labels,omitempty"`
}

type Backup struct {
	BucketName               string `yaml:"bucketName,omitempty"`
	DatabaseAdminServiceName string `yaml:"databaseAdminServiceName,omitempty"`
	DatabaseAdminServiceIP   string `yaml:"databaseAdminServiceIP,omitempty"`
	DatabaseNamespace        string `yaml:"databaseNamespace,omitempty" default:"default"`
	DatabaseBackupPort       string `yaml:"databaseBackupPort,omitempty" default:"6362"`
	DatabaseClusterDomain    string `yaml:"databaseClusterDomain,omitempty" default:"cluster.local"`
	Database                 string `yaml:"database,omitempty"`
	CloudProvider            string `yaml:"cloudProvider,omitempty"`
	SecretName               string `yaml:"secretName,omitempty"`
	SecretKeyName            string `yaml:"secretKeyName,omitempty"`
	PageCache                string `yaml:"pageCache,omitempty"`
	HeapSize                 string `yaml:"heapSize,omitempty"`
	FallbackToFull           bool   `yaml:"fallbackToFull" default:"true"`
	RemoveExistingFiles      bool   `yaml:"removeExistingFiles" default:"true"`
	RemoveBackupFiles        bool   `yaml:"removeBackupFiles" default:"true"`
	IncludeMetadata          string `yaml:"includeMetadata,omitempty"`
	Type                     string `yaml:"type,omitempty"`
	KeepFailed               bool   `yaml:"keepFailed" default:"false"`
	ParallelRecovery         bool   `yaml:"parallelRecovery" default:"false"`
	Verbose                  bool   `yaml:"verbose" default:"true"`
	CheckIndexes             bool   `yaml:"checkIndexes" default:"true"`
	CheckIndexStructure      bool   `yaml:"checkIndexStructure" default:"true"`
	CheckGraph               bool   `yaml:"checkGraph" default:"true"`
	CheckConsistency         bool   `yaml:"checkConsistency" default:"false"`
	PrepareRestore           bool   `yaml:"prepareRestore" default:"false"`
}
