#!/bin/bash

# Get the system architecture
ARCH=$(uname -m)

# Map the system architecture to the corresponding one used by the GitHub repository
case $ARCH in
    "x86_64")
        ARCH="amd64"
        ;;
    "aarch64")
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Get the latest release version and trim the leading 'v'
VERSION=$(curl -s https://api.github.com/repos/caas-team/sparrow/releases/latest | grep 'tag_name' | cut -d\" -f4 | tr -d v)

# Construct the download URL
URL="https://github.com/caas-team/sparrow/releases/download/v${VERSION}/sparrow_${VERSION}_linux_${ARCH}.tar.gz"

# Download the binary
curl -L $URL -o "sparrow_${VERSION}_linux_${ARCH}.tar.gz"

echo "Downloaded sparrow for $ARCH, version $VERSION"

# Unpack the binary
tar -xzf "sparrow_${VERSION}_linux_${ARCH}.tar.gz"

echo "Unpacked 'sparrow'"
