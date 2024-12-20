#!/bin/bash
set -eux
go run ./cmd/drawframe -cpuprofile cpu.prof "$1"
pprof -http=localhost:9999 cpu.prof