package operations

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// CheckConnectivity checks if there is connectivity with the provided kubernetes service or not
func CheckConnectivity(hostname string) error {
	ports := []string{"7474", "7687"}
	for _, port := range ports {
		hostPort := fmt.Sprintf("%s:%s", hostname, port)
		output, err := exec.Command("nc", "-vz", "-w", "3", hostname, port).CombinedOutput()
		if err != nil {
			return fmt.Errorf("connectivity cannot be established with %s \n output = %s \n err = %v",
				hostPort,
				string(output),
				err)
		}
		outputString := strings.ToLower(string(output))
		if !strings.Contains(outputString, "open") && !strings.Contains(outputString, "succeeded") {
			return fmt.Errorf("connectivity cannot be established with %s. Missing 'open' in output \n output = %s",
				hostPort,
				string(output))
		}
		log.Printf("Connectivity established with Service %s!!", hostPort)
	}

	return nil
}

// CheckEnvVariables checks if the environment variables required are present or not
func CheckEnvVariables() []error {
	envVarNames := []string{"SERVICE_NAME", "NAMESPACE", "DOMAIN", "PORT"}
	_, isIPPresent := os.LookupEnv("IP")
	var errs []error
	for _, name := range envVarNames {
		_, present := os.LookupEnv(name)
		if !present {
			switch name {
			case "DOMAIN":
				os.Setenv("DOMAIN", "cluster.local")
				continue
			case "NAMESPACE":
				os.Setenv("NAMESPACE", "default")
				continue
			default:
				if isIPPresent && name == "SERVICE_NAME" {
					continue
				}
				errs = append(errs, fmt.Errorf(" Missing %s environment variable !! ", name))
			}
		}
	}
	if len(errs) != 0 {
		return errs
	}
	return nil
}
