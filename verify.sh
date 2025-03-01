#!/bin/bash

go mod tidy
goimports -w .
golangci-lint run

export SHOULD_TEST_NETWORK=1
export LOCAL_FRAME_DIFFS=1

# Count=1 ensures no cached test results, which we want because some tests rely on networking results which
# can never be cached as they rely other computers to pass/fail.
go test -count=1 -race ./...
if [[ "$1" == "update" ]]; then
    find . -name '*frame.actual' -exec bash -c 'mv -f $0 ${0/frame.actual/frame}; echo "updating $0"' {} \;
fi