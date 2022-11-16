# index_updater

This directory contains Go code written to automate the index.yaml update during a helm chart release

## Building the executable

Run the following command to create an indexUpdater executable:

- The command below needs to be executed from inside the build/index_updater directory
```
- env GOOS=linux GOARCH=amd64 go build -o indexUpdater_linux main/main.go
```
- The above command creates an executable which works on TeamCity agents (linux based)

## Assumptions
- NEO4JVERSION and BRANCH environment variables must be set in your team city build
- indexUpdater_linux executable is executed from the root (helm-charts) directory. Based on this assumption it is able to read the index.yaml file
