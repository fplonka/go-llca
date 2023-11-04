#!/bin/bash

# Convenience script tested on ARM MacOS. Don't expect it to run on anything else.

# Exit on any non-zero status.
set -e

# ARM macOS compilation
echo "Compiling for ARM macOS..."
env GOOS=darwin GOARCH=arm64 go build -pgo default.pgo -o bin/go-llca_darwin_arm64

# x86 macOS compilation
echo "Compiling for x86 macOS..."
CGO_ENABLED=1 env GOOS=darwin GOARCH=amd64 go build -pgo default.pgo -o bin/go-llca_darwin_amd64

# x86 Linux compilation
# nevermind, couldn't get this to work
# echo "Compiling for x86 Linux..."
# env GOOS=linux GOARCH=amd64 go build -o bin/x86-linux ./...

# x86 Windows compilation
echo "Compiling for x86 Windows..."
CGO_ENABLED=1 env GOOS=windows GOARCH=amd64 go build -pgo default.pgo -o bin/go-llca_windows_amd64.exe
echo "Compilation finished."
