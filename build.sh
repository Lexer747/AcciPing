#!/bin/bash

SUPPORTED_TUPLES=("darwin amd64" "darwin arm64" "linux amd64" "linux arm64" "windows amd64")
OUT_DIR="out"
rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"
pushd "$OUT_DIR" &> /dev/null || exit
for i in "${SUPPORTED_TUPLES[@]}"; do
	set -- $i
	echo "Building	GOOS=$1	GOARCH=$2"
	mkdir -p "$1/$2" &> /dev/null
	pushd "$1/$2" &> /dev/null || exit
	env GOOS=$1 GOARCH=$2 go build github.com/Lexer747/AcciPing
	chmod +x AcciPing*
	popd &> /dev/null || exit
done
popd &> /dev/null
tree "$OUT_DIR"