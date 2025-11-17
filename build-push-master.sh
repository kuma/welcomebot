#!/bin/bash

# welcomebot Master Bot - Build and Push Container Script
# Builds multi-platform containers and pushes to Harbor registry

set -e  # Exit on error

# Configuration
REGISTRY="harbor.morikuma.org"
IMAGE_NAME="welcomebot/welcomebot-master"
TAG="latest"
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"
# Build for both Intel and ARM architectures
PLATFORMS="linux/amd64,linux/arm64"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== welcomebot Master Bot - Container Build & Push ===${NC}"
echo ""
echo "Configuration:"
echo "  Registry: ${REGISTRY}"
echo "  Image: ${IMAGE_NAME}"
echo "  Tag: ${TAG}"
echo "  Full Image: ${FULL_IMAGE}"
echo "  Platforms: ${PLATFORMS}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}✗ Error: Docker is not running${NC}"
    exit 1
fi

# Check if buildx is available
if ! docker buildx version > /dev/null 2>&1; then
    echo -e "${RED}✗ Error: Docker buildx is not available${NC}"
    echo "Please upgrade Docker to a version that supports buildx"
    exit 1
fi

# Create or use existing buildx builder
echo -e "${BLUE}=== Setting up buildx builder ===${NC}"
if ! docker buildx inspect multiarch > /dev/null 2>&1; then
    docker buildx create --name multiarch --use
else
    docker buildx use multiarch
fi

echo -e "${GREEN}✓ Buildx builder ready${NC}"
echo ""

# Login to Harbor registry
echo -e "${BLUE}=== Logging in to Harbor Registry ===${NC}"
echo "Please enter your Harbor credentials when prompted"
docker login ${REGISTRY}

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Login failed${NC}"
    exit 1
fi

# Build and push multi-platform image
echo ""
echo -e "${BLUE}=== Building and Pushing Multi-Platform Image ===${NC}"
echo "This will build for: ${PLATFORMS}"
echo ""

docker buildx build \
    --platform ${PLATFORMS} \
    -t ${FULL_IMAGE} \
    -f Dockerfile \
    --push \
    .

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ Successfully built and pushed ${FULL_IMAGE}${NC}"
    echo ""
    echo "Multi-platform image available for:"
    echo "  - linux/amd64 (Intel/AMD servers, Intel Macs)"
    echo "  - linux/arm64 (ARM servers, Apple Silicon Macs)"
    echo ""
    echo "You can now pull this image with:"
    echo "  docker pull ${FULL_IMAGE}"
else
    echo -e "${RED}✗ Build/Push failed${NC}"
    exit 1
fi

