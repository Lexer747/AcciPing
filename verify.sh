#!/bin/bash

go mod tidy
goimports -w .
golangci-lint run
# Count=1 ensures no cached test results
go test -count=1 ./...