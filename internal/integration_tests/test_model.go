package integration_tests

import (
	"strings"
	"time"
)

var (
	TestRunIdentifier string
)

func init() {
	dt := time.Now()
	dateTag := dt.Format("15:04:05 Mon")
	dateTag = strings.ReplaceAll(dateTag, " ", "-")
	dateTag = strings.ReplaceAll(dateTag, ":", "-")
	TestRunIdentifier = strings.ToLower(dateTag)
}
