package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

const jvmKey = "server.jvm.additional"

type setting struct {
	key   string
	value string
}

func chartOverrides(edition string) []setting {
	overrides := []setting{
		// Volume paths and connector settings are controlled by chart templates and user config.
		{key: "server.directories.import"},
		{key: "server.bolt.enabled"},
		{key: "server.http.enabled"},
		{key: "server.https.enabled"},
		// Preserve mounted log configs and existing operational defaults.
		{key: "server.logs.config", value: "/config/server-logs.xml/server-logs.xml"},
		{key: "server.logs.user.config", value: "/config/user-logs.xml/user-logs.xml"},
	}
	if edition == "enterprise" {
		return append(overrides,
			setting{key: "server.metrics.csv.rotation.compression", value: "zip"},
			setting{key: "server.panic.shutdown_on_panic", value: "true"},
		)
	}
	return append(overrides, setting{key: "db.tx_log.rotation.retention_policy", value: "1 days"})
}

func activeSettings(text string) (map[string][]string, error) {
	settings := make(map[string][]string)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !found || key == "" {
			return nil, fmt.Errorf("invalid configuration line: %s", line)
		}
		if _, exists := settings[key]; exists && key != jvmKey {
			return nil, fmt.Errorf("duplicate configuration key: %s", key)
		}
		settings[key] = append(settings[key], value)
	}
	if len(settings[jvmKey]) == 0 {
		return nil, fmt.Errorf("no default JVM arguments found")
	}
	return settings, nil
}

func withChartOverrides(text, edition string) string {
	overrides := chartOverrides(edition)
	byKey := make(map[string]string, len(overrides))
	for _, override := range overrides {
		byKey[override.key] = override.value
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		key, _, found := strings.Cut(strings.TrimSpace(line), "=")
		if value, overridden := byKey[strings.TrimSpace(key)]; found && overridden {
			if value == "" {
				lines = append(lines, "#"+line)
			}
			continue
		}
		lines = append(lines, line)
	}
	for _, override := range overrides {
		if override.value != "" {
			lines = append(lines, override.key+"="+override.value)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func compareConfigs(expected, actual string) (string, error) {
	want, err := activeSettings(expected)
	if err != nil {
		return "", err
	}
	got, err := activeSettings(actual)
	if err != nil {
		return "", err
	}
	keys := make([]string, 0, len(want)+len(got))
	for key := range want {
		keys = append(keys, key)
	}
	for key := range got {
		if _, exists := want[key]; !exists {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var difference strings.Builder
	for _, key := range keys {
		if slices.Equal(want[key], got[key]) {
			continue
		}
		for _, value := range want[key] {
			fmt.Fprintf(&difference, "- %s=%s\n", key, value)
		}
		for _, value := range got[key] {
			fmt.Fprintf(&difference, "+ %s=%s\n", key, value)
		}
	}
	return difference.String(), nil
}

func run(args []string, root, edition, image string, stdout io.Writer) error {
	if len(args) != 1 || (args[0] != "check" && args[0] != "update") {
		return fmt.Errorf("usage: neo4j-conf check|update")
	}
	if edition != "community" && edition != "enterprise" {
		return fmt.Errorf("NEO4J_EDITION must be community or enterprise")
	}
	if image == "" {
		return fmt.Errorf("NEO4J_DOCKER_IMG is required")
	}
	command := exec.Command("docker", "run", "--rm", "--entrypoint", "cat", image, "/var/lib/neo4j/conf/neo4j.conf")
	upstream, err := command.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("read neo4j.conf from %s: %w\n%s", image, err, exitError.Stderr)
		}
		return fmt.Errorf("read neo4j.conf from %s: %w", image, err)
	}
	if _, err := activeSettings(string(upstream)); err != nil {
		return fmt.Errorf("invalid config from %s: %w", image, err)
	}
	expected := withChartOverrides(string(upstream), edition)
	target := filepath.Join(root, "neo4j", "neo4j-"+edition+".conf")
	if args[0] == "update" {
		if err := os.WriteFile(target, []byte(expected), 0644); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Updated %s from %s\n", target, image)
		return nil
	}
	actual, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	difference, err := compareConfigs(expected, string(actual))
	if err != nil {
		return fmt.Errorf("compare %s: %w", target, err)
	}
	if difference != "" {
		return fmt.Errorf("configuration drift between %s (-) and %s (+):\n%sRefresh with bin/update-neo4j-conf using the same NEO4J_EDITION and NEO4J_DOCKER_IMG", image, target, difference)
	}
	fmt.Fprintf(stdout, "Neo4j %s defaults match %s\n", edition, image)
	return nil
}

func main() {
	root, err := os.Getwd()
	if err == nil {
		err = run(os.Args[1:], root, os.Getenv("NEO4J_EDITION"), os.Getenv("NEO4J_DOCKER_IMG"), os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
