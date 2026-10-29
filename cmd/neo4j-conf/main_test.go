package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const upstreamConfig = `# Neo4j defaults
server.jvm.additional=--add-opens=java.base/java.util.concurrent=ALL-UNNAMED
server.jvm.additional=--enable-native-access=ALL-UNNAMED
db.query.default_language=CYPHER_25
server.http.enabled=true
server.directories.import=import
`

func TestComparisonIgnoresCommentsWhitespaceAndKeyOrder(t *testing.T) {
	changed := strings.ReplaceAll(upstreamConfig, "# Neo4j defaults", "# Different comment")
	changed = strings.ReplaceAll(changed, "db.query.default_language=CYPHER_25", " db.query.default_language = CYPHER_25 ")
	changed = strings.ReplaceAll(changed, "server.http.enabled=true\n", "") + "server.http.enabled=true\n"
	difference, err := compareConfigs(upstreamConfig, changed)
	if err != nil || difference != "" {
		t.Fatalf("expected equivalent settings, got difference %q and error %v", difference, err)
	}
}

func TestJvmArgumentsPreserveValuesOrderAndDuplicates(t *testing.T) {
	settings, err := activeSettings(upstreamConfig)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--add-opens=java.base/java.util.concurrent=ALL-UNNAMED",
		"--enable-native-access=ALL-UNNAMED",
	}
	if !slices.Equal(settings[jvmKey], want) {
		t.Fatalf("expected JVM arguments %v, got %v", want, settings[jvmKey])
	}
	for _, changed := range []string{
		upstreamConfig + "server.jvm.additional=--enable-native-access=ALL-UNNAMED\n",
		strings.ReplaceAll(upstreamConfig, strings.Join(want, "\nserver.jvm.additional="), strings.Join([]string{want[1], want[0]}, "\nserver.jvm.additional=")),
	} {
		if difference, err := compareConfigs(upstreamConfig, changed); err != nil || difference == "" {
			t.Fatalf("expected JVM argument drift, got difference %q and error %v", difference, err)
		}
	}
}

func TestInvalidConfigIsRejected(t *testing.T) {
	for _, text := range []string{
		"", "# Only comments", "not a property", "=missing key",
		upstreamConfig + "server.http.enabled=false\n",
	} {
		t.Run(text, func(t *testing.T) {
			if _, err := activeSettings(text); err == nil {
				t.Fatal("expected invalid configuration to be rejected")
			}
		})
	}
}

func TestChartOverrides(t *testing.T) {
	for _, edition := range []string{"community", "enterprise"} {
		t.Run(edition, func(t *testing.T) {
			text := withChartOverrides(upstreamConfig, edition)
			if again := withChartOverrides(text, edition); again != text {
				t.Fatal("applying chart overrides twice changed the config")
			}
			settings, err := activeSettings(text)
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]string{
				"server.logs.config":      "/config/server-logs.xml/server-logs.xml",
				"server.logs.user.config": "/config/user-logs.xml/user-logs.xml",
			}
			if edition == "enterprise" {
				want["server.panic.shutdown_on_panic"] = "true"
				want["server.metrics.csv.rotation.compression"] = "zip"
			} else {
				want["db.tx_log.rotation.retention_policy"] = "1 days"
				if _, found := settings["server.panic.shutdown_on_panic"]; found {
					t.Fatal("enterprise-only setting added to community config")
				}
			}
			for key, value := range want {
				if !slices.Equal(settings[key], []string{value}) {
					t.Errorf("expected %s=%s, got %v", key, value, settings[key])
				}
			}
			for _, key := range []string{"server.http.enabled", "server.directories.import"} {
				if _, found := settings[key]; found {
					t.Errorf("setting %s should be controlled by chart templates and user config", key)
				}
			}
		})
	}
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func fakeDocker(t *testing.T, config string) string {
	t.Helper()
	directory := t.TempDir()
	source := filepath.Join(directory, "neo4j.conf")
	writeTestFile(t, source, config, 0600)
	writeTestFile(t, filepath.Join(directory, "docker"), `#!/bin/sh
printf '%s\n' "$@" > "$NEO4J_CONF_TEST_ARGS"
if [ "${NEO4J_CONF_TEST_FAIL:-}" = 1 ]
then
  echo 'image unavailable' >&2
  exit 1
fi
cat "$NEO4J_CONF_TEST_SOURCE"
`, 0700)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NEO4J_CONF_TEST_SOURCE", source)
	t.Setenv("NEO4J_CONF_TEST_ARGS", filepath.Join(directory, "args"))
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "neo4j"), 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCheckDetectsConfigurationDrift(t *testing.T) {
	root := fakeDocker(t, upstreamConfig)
	expected := withChartOverrides(upstreamConfig, "community")
	target := filepath.Join(root, "neo4j", "neo4j-community.conf")
	for _, test := range []struct {
		name   string
		config string
		drift  bool
	}{
		{name: "matching", config: expected},
		{name: "missing JVM argument", config: strings.ReplaceAll(expected, "server.jvm.additional=--enable-native-access=ALL-UNNAMED\n", ""), drift: true},
		{name: "changed default", config: strings.ReplaceAll(expected, "CYPHER_25", "CYPHER_5"), drift: true},
		{name: "extra setting", config: expected + "unexpected.setting=true\n", drift: true},
		{name: "changed chart override", config: strings.ReplaceAll(expected, "/config/server-logs.xml/server-logs.xml", "wrong.xml"), drift: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			writeTestFile(t, target, test.config, 0600)
			var output bytes.Buffer
			err := run([]string{"check"}, root, "community", "neo4j:2026.10.1", &output)
			if (err != nil) != test.drift {
				t.Fatalf("expected drift=%t, got %v", test.drift, err)
			}
			if test.drift && !strings.Contains(err.Error(), "configuration drift") {
				t.Fatalf("expected actionable drift error, got %v", err)
			}
			actual, readErr := os.ReadFile(target)
			if readErr != nil || string(actual) != test.config {
				t.Fatalf("check changed the config: %v", readErr)
			}
		})
	}
}

func TestFailedImageReadDoesNotOverwriteConfig(t *testing.T) {
	for _, failure := range []string{"empty config", "docker failure"} {
		t.Run(failure, func(t *testing.T) {
			root := fakeDocker(t, "")
			if failure == "docker failure" {
				t.Setenv("NEO4J_CONF_TEST_FAIL", "1")
			}
			target := filepath.Join(root, "neo4j", "neo4j-enterprise.conf")
			writeTestFile(t, target, "original config", 0600)
			var output bytes.Buffer
			if err := run([]string{"update"}, root, "enterprise", "neo4j:2026.10.1-enterprise", &output); err == nil {
				t.Fatal("expected image read to fail")
			}
			actual, err := os.ReadFile(target)
			if err != nil || string(actual) != "original config" {
				t.Fatalf("failed image read changed the config: %v", err)
			}
		})
	}
}

func TestUpdateUsesRequestedImage(t *testing.T) {
	root := fakeDocker(t, upstreamConfig)
	var output bytes.Buffer
	if err := run([]string{"update"}, root, "enterprise", "neo4j:2026.10.1-enterprise", &output); err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(os.Getenv("NEO4J_CONF_TEST_ARGS"))
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := "run\n--rm\n--entrypoint\ncat\nneo4j:2026.10.1-enterprise\n/var/lib/neo4j/conf/neo4j.conf\n"
	if string(args) != wantArgs {
		t.Fatalf("unexpected Docker arguments: %s", args)
	}
	actual, err := os.ReadFile(filepath.Join(root, "neo4j", "neo4j-enterprise.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if want := withChartOverrides(upstreamConfig, "enterprise"); string(actual) != want {
		t.Fatalf("unexpected updated config: %s", actual)
	}
}
