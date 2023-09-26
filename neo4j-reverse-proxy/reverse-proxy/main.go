package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reverse-proxy/operations"
	"reverse-proxy/proxy"
)

func main() {

	startup()

	h, err := proxy.NewHandle()
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", h)

	domain := fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT"))
	log.Printf("Listening on %s", domain)

	log.Fatal(http.ListenAndServe(domain, nil))
}

func startup() {
	errors := operations.CheckEnvVariables()
	if len(errors) != 0 {
		log.Fatalf("%v", errors)
	}

	hostname := fmt.Sprintf("%s.%s.svc.%s", os.Getenv("SERVICE_NAME"), os.Getenv("NAMESPACE"), os.Getenv("DOMAIN"))
	if ip, present := os.LookupEnv("IP"); present {
		hostname = ip
	}

	err := operations.CheckConnectivity(hostname)
	if err != nil {
		log.Fatal(err)
	}
}
