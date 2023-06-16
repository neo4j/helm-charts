package model

import (
	"fmt"
	"os"
	"strings"
)

type ReleaseName interface {
	String() string
	Namespace() Namespace
	DiskName() PersistentDiskName
	PodName() string
	EnvConfigMapName() string
	UserConfigMapName() string
	InternalServiceName() string
	DefaultConfigMapName() string
}

func NewReleaseName(name string) ReleaseName {
	r := releaseName(name)
	return &r
}

type releaseName string

func (r *releaseName) String() string {
	return string(*r)
}

func (r *releaseName) Namespace() Namespace {
	return Namespace("neo4j-" + string(*r))
}

func (r *releaseName) DiskName() PersistentDiskName {
	return PersistentDiskName(fmt.Sprintf("neo4j-data-disk-%s", *r))
}

func (r *releaseName) PodName() string {
	return string(*r) + "-0"
}

func (r *releaseName) EnvConfigMapName() string {
	return string(*r) + "-env"
}

func (r *releaseName) DefaultConfigMapName() string {
	return string(*r) + "-default-config"
}

func (r *releaseName) UserConfigMapName() string {
	return string(*r) + "-user-config"
}

func (r *releaseName) InternalServiceName() string {
	return string(*r) + "-internals"
}

func NewCoreReleaseName(clusterName ReleaseName, number int) ReleaseName {
	r := clusterMemberReleaseName{clusterName, releaseName(fmt.Sprintf("%s-core-%d", clusterName, number))}
	return &r
}

func NewReadReplicaReleaseName(clusterName ReleaseName, number int) ReleaseName {
	r := clusterMemberReleaseName{clusterName, releaseName(fmt.Sprintf("%s-read-replica-%d", clusterName, number))}
	return &r
}

func NewLoadBalancerReleaseName(clusterName ReleaseName) ReleaseName {
	r := clusterMemberReleaseName{clusterName, releaseName(fmt.Sprintf("%s-loadbalancer", clusterName))}
	return &r
}

func NewHeadlessServiceReleaseName(clusterName ReleaseName) ReleaseName {
	r := clusterMemberReleaseName{clusterName, releaseName(fmt.Sprintf("%s-headless", clusterName))}
	return &r
}

type clusterMemberReleaseName struct {
	clusterName ReleaseName
	memberName  releaseName
}

func (r *clusterMemberReleaseName) String() string {
	return r.memberName.String()
}

func (r *clusterMemberReleaseName) Namespace() Namespace {
	return r.clusterName.Namespace()
}

func (r *clusterMemberReleaseName) DiskName() PersistentDiskName {
	return r.memberName.DiskName()
}

func (r *clusterMemberReleaseName) PodName() string {
	return r.memberName.PodName()
}

func (r *clusterMemberReleaseName) EnvConfigMapName() string {
	return r.memberName.EnvConfigMapName()
}

func (r *clusterMemberReleaseName) UserConfigMapName() string {
	return r.memberName.UserConfigMapName()
}

func (r *clusterMemberReleaseName) InternalServiceName() string {
	return r.memberName.InternalServiceName()
}

func (r *clusterMemberReleaseName) DefaultConfigMapName() string {
	return r.memberName.DefaultConfigMapName()
}

type Namespace string
type PersistentDiskName string

var DefaultEnterpriseValues = HelmValues{
	Neo4J: Neo4J{
		Name:                   "test",
		AcceptLicenseAgreement: "yes",
		Edition:                "enterprise",
	},
	Volumes: Volumes{
		Data: Data{
			Mode:           "selector",
			DisableSubPath: false,
		},
	},
}

var DefaultCommunityValues = HelmValues{
	Neo4J: Neo4J{
		Name:    "test",
		Edition: "community",
	},
	Volumes: Volumes{
		Data: Data{
			Mode:           "selector",
			DisableSubPath: false,
		},
	},
}

var DefaultNeo4jBackupValues = Neo4jBackupValues{

	Neo4J: Neo4jBackupNeo4j{
		Image:                      strings.Split(os.Getenv("NEO4J_DOCKER_BACKUP_IMG"), ":")[0],
		ImageTag:                   strings.Split(os.Getenv("NEO4J_DOCKER_BACKUP_IMG"), ":")[1],
		JobSchedule:                "* * * * *",
		SuccessfulJobsHistoryLimit: 3,
		FailedJobsHistoryLimit:     1,
		BackoffLimit:               6,
	},
	Backup: Backup{
		CheckIndexes:        true,
		CheckIndexStructure: true,
		CheckGraph:          true,
		CheckConsistency:    true,
		PrepareRestore:      true,
	},
	TempVolume: map[string]interface{}{
		"emptyDir": nil,
	},
	SecurityContext: SecurityContext{
		RunAsNonRoot:        true,
		RunAsUser:           7474,
		RunAsGroup:          7474,
		FsGroup:             7474,
		FsGroupChangePolicy: "Always",
	},
}
