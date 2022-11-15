package main

import (
	"github.com/neo4j/helm-charts/build/index_updater"
	"log"
)

func main() {
	err := index_updater.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
