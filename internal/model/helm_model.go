package model

import "fmt"

type ReleaseName string

func (r *ReleaseName) Namespace() Namespace {
	return Namespace("neo4j-" + string(*r))
}

func (r *ReleaseName) DiskName() PersistentDiskName {
	return PersistentDiskName(fmt.Sprintf("neo4j-data-disk-%s", *r))
}

func (r *ReleaseName) PodName() string {
	return string(*r) + "-0"
}

func (r *ReleaseName) EnvConfigMapName() string {
	return string(*r) + "-env"
}

func (r *ReleaseName) UserConfigMapName() string {
	return string(*r) + "-user-config"
}

type Namespace string
type PersistentDiskName string
