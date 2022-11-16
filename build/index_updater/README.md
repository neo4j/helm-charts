# index_updater/

This directory contains Go code written to automate the index.Yaml update during a helm chart release

## Building the executable

Run the following command to create an indexUpdater executable:

- The command needs to be executed from inside the build/index_updater directory
```
- env GOOS=linux GOARCH=amd64 go build -o indexUpdater_linux main/main.go
```
- The above command creates an executable which works on TeamCity agents (linux based)
