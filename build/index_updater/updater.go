package index_updater

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

func getSha256Sum(packageName string) (string, error) {
	log.Printf("Calculating sha256sum for %s", packageName)
	cmd := exec.Command("sha256sum", packageName)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	sha := strings.Split(out.String(), " ")
	return strings.TrimSpace(sha[0]), nil
}

func getHelmPackages(packageName string, version string) error {
	log.Printf("Downloading %s-%s", packageName, version)
	cmd := exec.Command("helm", "pull", packageName, "--version", version)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func getCurrentDateAndTime() string {
	return time.Now().Format("2006-01-02T15:04:05.000000000")
}

// getNewChartEntries returns the latest helm charts details as a list of entries
func getNewChartEntries() []*Entry {
	var newEntries []*Entry

	for _, chart := range chartsList {

		chartPath := fmt.Sprintf("%s/%s", helmRepo, chart)

		err := getHelmPackages(chartPath, version)
		if err != nil {
			log.Fatalf(fmt.Sprintf("Error while downloading package %s version %s \n %v", chartPath, version, err))
		}

		sha, err := getSha256Sum(fmt.Sprintf("%s-%s.tgz", chart, version))
		if err != nil {
			log.Fatalf(fmt.Sprintf("Error while calculating sha256sum for %s version %s \n %v", chartPath, version, err))
		}

		newEntries = append(newEntries, NewEntry(sha, chart, branch))
	}
	return newEntries
}

func Execute() error {

	indexYaml, err := NewIndexYaml()
	if err != nil {
		return err
	}

	newEntries := getNewChartEntries()

	indexYaml.UpdateEntries(newEntries)

	err = updateIndexYaml(&indexYaml)
	if err != nil {
		return err
	}

	return nil
}
