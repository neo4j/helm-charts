package unit_tests

import (
	"fmt"
	"strings"
)

var funcMap = map[string]func(cpuValue string) string{
	"cpuRequests": func(cpuValue string) string {
		return fmt.Sprintf("neo4j.resources.requests.cpu=%s", cpuValue)
	},
	"memoryRequests": func(memValue string) string {
		return fmt.Sprintf("neo4j.resources.requests.memory=%s", memValue)
	},
	"memoryResources": func(memValue string) string {
		return fmt.Sprintf("neo4j.resources.memory=%s", memValue)
	},
	"cpuResources": func(cpuValue string) string {
		return fmt.Sprintf("neo4j.resources.cpu=%s", cpuValue)
	},
}

type Neo4jResourceTestCase struct {
	arguments []string
	cpu       string
	memory    string
}

func GenerateNeo4jResourcesTestCase(funcs []string, cpu string, memory string) Neo4jResourceTestCase {
	if cpu == "" {
		cpu = "1"
	}
	if memory == "" {
		memory = "2Gi"
	}
	return Neo4jResourceTestCase{
		arguments: getArgs(funcs, cpu, memory),
		cpu:       cpu,
		memory:    memory,
	}
}

func getArgs(funcs []string, cpu string, memory string) []string {

	args := []string{"--set", "volumes.data.mode=selector", "--set", "neo4j.acceptLicenseAgreement=yes"}
	for _, funcName := range funcs {
		f := funcMap[funcName]
		if strings.Contains(funcName, "cpu") {
			args = append(args, "--set", f(cpu))
		}
		if strings.Contains(funcName, "memory") {
			args = append(args, "--set", f(memory))
		}
	}
	return args
}
