package index_updater

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

func readIndexYaml() ([]byte, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/index.yaml", dir)
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func updateIndexYaml(indexYaml *IndexYaml) error {

	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	yamlEncoder.Encode(&indexYaml)

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/index.yaml", dir)
	err = os.WriteFile(path, b.Bytes(), 0)
	if err != nil {
		return err
	}
	fmt.Println("index.yaml updated !!")
	return nil
}
