package model

import (
	"fmt"
)

type ReleaseName interface {
	String() string
	Namespace() Namespace
	DiskName() PersistentDiskName
	PodName() string
	EnvConfigMapName() string
	UserConfigMapName() string
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

func (r *releaseName) UserConfigMapName() string {
	return string(*r) + "-user-config"
}

func NewCoreReleaseName(clusterName ReleaseName, number int) ReleaseName {
	r := clusterMemberReleaseName{clusterName, releaseName(fmt.Sprintf("%s-core-%d", clusterName, number))}
	return &r
}

func NewLoadBalancerReleaseName(clusterName ReleaseName) ReleaseName {
	r := clusterMemberReleaseName{clusterName, releaseName(fmt.Sprintf("%s-loadbalancer", clusterName))}
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

type Namespace string
type PersistentDiskName string
