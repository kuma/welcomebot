#!/bin/bash
# Production Reload Script
# Rebuilds images, pushes to Harbor, and restarts production deployments

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🔄 Reloading Production (welcomebot-lightprod)${NC}"
echo ""

# Configuration - Harbor settings
HARBOR_REGISTRY="${HARBOR_REGISTRY:-harbor.morikuma.org}"
HARBOR_PROJECT="${HARBOR_PROJECT:-welcomebot}"
HARBOR_USERNAME="${HARBOR_USERNAME:-admin}"
HARBOR_PASSWORD="${HARBOR_PASSWORD:-!Containerisno1}"
IMAGE_TAG="${IMAGE_TAG:-lightprod}"

# Verify we're in the correct directory
if [ ! -f "go.mod" ] || [ ! -d "cmd/master" ]; then
    echo -e "${RED}❌ Error: Must run from project root directory${NC}"
    exit 1
fi

# Login to Harbor
echo -e "${BLUE}🔐 Logging in to Harbor...${NC}"
echo "$HARBOR_PASSWORD" | docker login "$HARBOR_REGISTRY" -u "$HARBOR_USERNAME" --password-stdin > /dev/null 2>&1 || {
    echo -e "${RED}❌ Failed to login to Harbor${NC}"
    exit 1
}
echo -e "${GREEN}  ✓ Logged in to Harbor${NC}"
echo ""

# Setup Docker buildx for multi-architecture builds
echo -e "${BLUE}🔧 Setting up Docker buildx...${NC}"
docker buildx inspect multiarch-builder > /dev/null 2>&1 || {
    docker buildx create --name multiarch-builder --use > /dev/null 2>&1
}
docker buildx use multiarch-builder > /dev/null 2>&1
echo -e "${GREEN}  ✓ Buildx ready${NC}"
echo ""

# Build and push multi-architecture images (amd64 + arm64)
echo -e "${BLUE}📦 Rebuilding multi-arch images (amd64 + arm64)...${NC}"
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -t "$HARBOR_REGISTRY/$HARBOR_PROJECT/welcomebot-master:$IMAGE_TAG" \
    -t "$HARBOR_REGISTRY/$HARBOR_PROJECT/welcomebot-worker:$IMAGE_TAG" \
    --push \
    . || {
    echo -e "${RED}❌ Failed to build/push images${NC}"
    exit 1
}
echo -e "${GREEN}  ✓ Images rebuilt and pushed (master and worker)${NC}"
echo ""

# Restart deployments
echo -e "${BLUE}🔄 Restarting deployments...${NC}"

# Restart master
kubectl rollout restart deployment/welcomebot-master -n welcomebot-lightprod || {
    echo -e "${RED}❌ Failed to restart master${NC}"
    exit 1
}
echo -e "${GREEN}  ✓ Master restarting${NC}"

# Restart workers (10 slaves)
for i in {1..10}; do
    kubectl rollout restart deployment/welcomebot-worker-slave$i -n welcomebot-lightprod || {
        echo -e "${YELLOW}  ⚠ Failed to restart worker slave$i${NC}"
    }
done
echo -e "${GREEN}  ✓ Workers restarting${NC}"
echo ""

# Wait for rollouts
echo -e "${BLUE}⏳ Waiting for rollouts to complete...${NC}"
kubectl rollout status deployment/welcomebot-master -n welcomebot-lightprod --timeout=120s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Master rollout timeout (check logs)${NC}"
}
for i in {1..10}; do
    kubectl rollout status deployment/welcomebot-worker-slave$i -n welcomebot-lightprod --timeout=120s 2>/dev/null || {
        echo -e "${YELLOW}  ⚠ Worker slave$i rollout timeout (check logs)${NC}"
    }
done

echo ""
echo -e "${GREEN}✅ Reload complete!${NC}"
echo ""
echo -e "${BLUE}Current status:${NC}"
kubectl get pods -n welcomebot-lightprod
echo ""
echo -e "${BLUE}💡 Tip: View logs with ${YELLOW}./scripts/prod-logs.sh${NC}"

