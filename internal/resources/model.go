package resources

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

var _, thisFile, _, _ = runtime.Caller(0)
var resourcesDir = path.Dir(thisFile)

type YamlFile interface {
	Path() string
	HelmArgs() []string
	Data() (map[interface{}]interface{}, error)
}

type yamlFile struct {
	path string
}

func (y *yamlFile) Path() string {
	return y.path
}

func (y *yamlFile) HelmArgs() []string {
	return []string{"-f", y.path}
}

func (y *yamlFile) Data() (map[interface{}]interface{}, error) {
	file, err := ioutil.ReadFile(y.Path())
	if err != nil {
		return nil, err
	}
	data := make(map[interface{}]interface{})
	err = yaml.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func resourceExistsAt(path string) (bool, error) {
	if fileInfo, err := os.Stat(path); err == nil {
		if filepath.Ext(path) == ".yaml" && !fileInfo.IsDir() {
			return true, nil
		}
		return false, fmt.Errorf("unexpected error occured. File %s returned fileInfo: %v", path, fileInfo)
	} else {
		return false, err
	}
}

func newYamlFile(filename string) YamlFile {
	fullPath := path.Join(resourcesDir, filename)
	if exists, err := resourceExistsAt(fullPath); err != nil || !exists {
		panic(err)
	}
	return &yamlFile{fullPath}
}
