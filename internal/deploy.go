package internal

import (
	"bufio"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
)

// This changes the working directory to the parent directory if the current working directory doesn't contain a directory called "yaml"
func init() {
	_, filename, _, _ := runtime.Caller(0)
	currentDir := path.Dir(filename)
	files, err := ioutil.ReadDir(currentDir)
	CheckError(err)
	for _, file := range files {
		if file.Name() == "yaml" {
			return
		}
	}
	dir := path.Join(currentDir, "..")
	err = os.Chdir(dir)
	CheckError(err)

	os.Setenv("KUBECONFIG", path.Join(dir, ".kube/config"))
}

func helmInstallCommands() [][]string {
	imageArg := []string{}
	if value, found := os.LookupEnv("NEO4J_DOCKER_IMG"); found {
		imageArg = []string{"--set", "image.customImage=" + value}
	}
	return [][]string{
		append([]string{"install", "neo4j", "./neo4j", "--namespace", "neo4j", "--create-namespace", "--wait", "--timeout", "300s"}, imageArg...),
	}
}

var kSetupCommands = [][]string{
	{"apply", "-f", "yaml/neo4j-gce-storageclass.yaml"},  // it doesnt matter if this already exists currently and it's a PITA to clean up so just apply here
	{"create", "-f", "yaml/neo4j-persistentvolume.yaml"}, // create because if this already exists we run into problems (pv are not namespaced)
}

var helmCleanupCommands = [][]string{
	{"uninstall", "neo4j", "--namespace", "neo4j"},
}

var kCleanupCommands = [][]string{
	{"delete", "namespace", "neo4j", "--ignore-not-found"},
	{"delete", "persistentvolumes", "neo4j-data-storage"},
	{"delete", "storageclass", "neo4j-storage"},
}

type Closeable func() error

func proxyBolt() (Closeable, error) {

	cmd := exec.Command("kubectl", "--namespace", "neo4j", "port-forward", "service/neo4j-lb", "7474:7474", "7687:7687")
	stdout, err := cmd.StdoutPipe()
	CheckError(err)
	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to signal that we are ready
	started := make(chan struct{})

	// Create a scanner which scans in a line-by-line fashion
	scanner := bufio.NewScanner(stdout)

	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block
	go func() {
		var once sync.Once
		notifyStarted := func() { started <- struct{}{} }

		// We're all done, unblock the channel
		defer func() {
			once.Do(notifyStarted)
		}()

		// Read line by line and process it until we see that Forwarding has begun
		for scanner.Scan() {
			line := scanner.Text()
			log.Print("PortForward:", line)
			if strings.HasPrefix(line, "Forwarding from") {
				once.Do(notifyStarted)
			}
		}
		scannerErr := scanner.Err()
		if scannerErr != nil {
			log.Printf("Scanner logged error %s - this is usually expected when the proxy is terminated", scannerErr)
		}
	}()

	// Start the command and check for errors
	err = cmd.Start()
	if err == nil {
		// Wait for output to indicate we actually started forwarding
		<-started
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			err = fmt.Errorf("port forward process exited unexpectedly")
		}
	}

	return func() error {
		var cmdErr = cmd.Process.Kill()
		if cmdErr != nil {
			log.Print("failed to kill process: ", cmdErr)
		}
		stdout.Close()
		return cmdErr
	}, err
}

func InstallNeo4j(zone Zone, project Project) Closeable {

	err := run("gcloud", "container", "clusters", "get-credentials", string(CurrentCluster))
	CheckError(err)

	var closeables []Closeable
	addCloseable := func(closeable Closeable) {
		closeables = append([]Closeable{closeable}, closeables...)
	}

	cleanup := func() error {
		var combinedErrors error
		if closeables != nil {
			for _, closeable := range closeables {
				err := closeable()
				if err != nil {
					combinedErrors = combineErrors(combinedErrors, err)
				}
			}
		}
		return combinedErrors
	}

	completed := false
	defer func() (err error) {
		if !completed {
			err = cleanup()
			CheckError(err)
		}
		return err
	}()

	cleanupDisk, err := createDisk(zone, project)
	addCloseable(cleanupDisk)
	CheckError(err)

	addCloseable(func() error { return runAll("kubectl", kCleanupCommands, false) })
	err = runAll("kubectl", kSetupCommands, true)
	addCloseable(func() error { return runAll("helm", helmCleanupCommands, false) })
	err = runAll("helm", helmInstallCommands(), true)

	CheckError(err)

	completed = true
	return cleanup
}

func combineErrors(firstOrNil error, second error) error {
	if firstOrNil == nil {
		firstOrNil = second
	} else {
		firstOrNil = multierror.Append(firstOrNil, second)
	}
	return firstOrNil
}

func runAll(bin string, commands [][]string, failFast bool) error {
	var combinedErrors error
	for _, command := range commands {
		err := run(bin, command...)
		if err != nil {
			if failFast {
				return err
			} else {
				combinedErrors = combineErrors(combinedErrors, err)
			}
		}
	}
	return combinedErrors
}

func createDisk(zone Zone, project Project) (Closeable, error) {
	err := run("gcloud", "compute", "disks", "create", "--size", "10GB", "--type", "pd-ssd", "neo4j-data-disk", "--zone="+string(zone), "--project="+string(project))
	return func() error { return deleteDisk(zone, project) }, err
}

func deleteDisk(zone Zone, project Project) error {
	return run("gcloud", "compute", "disks", "delete", "neo4j-data-disk", "--zone="+string(zone), "--project="+string(project))
}

func run(command string, args ...string) error {
	log.Print("running: ", command, args)
	out, err := exec.Command(command, args...).CombinedOutput()
	if out != nil {
		fmt.Printf("output: %s\n", out)
	}
	return err
}
