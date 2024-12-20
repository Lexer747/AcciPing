#!/bin/bash

go mod tidy
goimports -w .
golangci-lint run
# Count=1 ensures no cached test results
go test -count=1 ./...
if [[ "$1" == "update" ]]; then
    find . -name '*frame.actual' -exec bash -c 'mv -f $0 ${0/frame.actual/frame}; echo "updating $0"' {} \;
fi