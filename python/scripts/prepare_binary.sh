#!/bin/bash

set -e  # Exit on error

mkdir -p sandboxai/bin

# Ensure the correct binary is copied to the target location
if [[ "$CIBW_ARCHS" == "x86_64" && "$CIBW_PLATFORM" == "linux" ]]; then
    echo "Using Linux x86_64 binary"
    cp bin/sandboxaid_linux_amd64_v1/sandboxaid sandboxai/bin/
elif [[ "$CIBW_ARCHS" == "aarch64" && "$CIBW_PLATFORM" == "linux" ]]; then
    echo "Using Linux aarch64 binary"
    cp bin/sandboxaid_linux_arm64_v8.0/sandboxaid sandboxai/bin/
elif [[ "$CIBW_ARCHS" == "arm64" && "$CIBW_PLATFORM" == "macos" ]]; then
    echo "Using macOS ARM64 binary"
    cp bin/sandboxaid_darwin_arm64_v8.0/sandboxaid sandboxai/bin/
elif [[ "$CIBW_ARCHS" == "x86_64" && "$CIBW_PLATFORM" == "macos" ]]; then
    echo "Using macOS x86_64 binary"
    cp bin/sandboxaid_darwin_amd64_v1/sandboxaid sandboxai/bin/
else
    echo "Unsupported platform: $CIBW_PLATFORM - $CIBW_ARCHS"
    exit 1
fi

# Confirm the file was copied
ls -l sandboxai/bin/sandboxaid