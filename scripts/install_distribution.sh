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

# Get the distribution of the system
DISTRIBUTION=$(lsb_release -is | tr '[:upper:]' '[:lower:]')

# Map the distribution to the corresponding package extension
case $DISTRIBUTION in
    "debian"|"ubuntu")
        EXT="deb"
        ;;
    "fedora"|"centos"|"rhel")
        EXT="rpm"
        ;;
    "alpine")
        EXT="apk"
        ;;
    *)
        echo "Unsupported distribution: $DISTRIBUTION"
        exit 1
        ;;
esac

# Construct the download URL
URL="https://github.com/caas-team/sparrow/releases/download/v${VERSION}/sparrow_${VERSION}_linux_${ARCH}.${EXT}"

# Download the binary
curl -L $URL -o "sparrow_${VERSION}_linux_${ARCH}.${EXT}"

echo "Downloaded sparrow for $ARCH, version $VERSION"

# Unpack and install the binary
case $EXT in
    "deb")
        sudo dpkg -i "sparrow_${VERSION}_linux_${ARCH}.${EXT}"
        ;;
    "rpm")
        sudo rpm -i "sparrow_${VERSION}_linux_${ARCH}.${EXT}"
        ;;
    "apk")
        sudo apk add "sparrow_${VERSION}_linux_${ARCH}.${EXT}"
        ;;
esac