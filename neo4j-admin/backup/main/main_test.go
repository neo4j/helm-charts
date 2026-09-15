package main

import (
	"log"
	"os"
	"testing"
)

func TestLogDestinations(t *testing.T) {
	originalOutput := log.Writer()
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
	})

	configureLogging()

	if log.Writer() != os.Stdout {
		t.Fatal("standard logs must be written to stdout")
	}
	if errorLogger.Writer() != os.Stderr {
		t.Fatal("fatal errors must be written to stderr")
	}
}
