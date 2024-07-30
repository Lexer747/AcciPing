#!/bin/bash

go mod tidy
goimports -w .
golangci-lint run
go test -count=1 ./...