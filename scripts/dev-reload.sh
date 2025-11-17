#!/bin/bash
# Quick Rebuild and Redeploy Script
# Rebuilds images and restarts deployments

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🔄 Reloading Discord Bot Local Development${NC}"
echo ""

# Verify we're in the correct directory
if [ ! -f "go.mod" ] || [ ! -d "cmd/master" ]; then
    echo -e "${RED}❌ Error: Must run from project root directory${NC}"
    exit 1
fi

# Check we're on docker-desktop context
CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null || echo "")

if [ "$CURRENT_CONTEXT" != "docker-desktop" ]; then
    echo -e "${YELLOW}⚠ Not on docker-desktop context (currently: $CURRENT_CONTEXT)${NC}"
    echo -e "${BLUE}Switching to docker-desktop...${NC}"
    kubectl config use-context docker-desktop || {
        echo -e "${RED}❌ Failed to switch to docker-desktop context${NC}"
        exit 1
    }
fi

CLUSTER_TYPE="docker-desktop"

# Build Docker images
echo -e "${BLUE}📦 Rebuilding Docker images...${NC}"
docker build -t welcomebot-master:local . > /dev/null 2>&1 || {
    echo -e "${RED}❌ Failed to build master image${NC}"
    docker build -t welcomebot-master:local .
    exit 1
}
docker build -t welcomebot-worker:local . > /dev/null 2>&1 || {
    echo -e "${RED}❌ Failed to build worker image${NC}"
    docker build -t welcomebot-worker:local .
    exit 1
}
echo -e "${GREEN}  ✓ Images rebuilt${NC}"
echo ""

# Docker Desktop uses local Docker daemon - images already available
echo -e "${BLUE}📥 Images ready${NC}"
echo ""

# Restart deployments
echo -e "${BLUE}🔄 Restarting deployments...${NC}"
kubectl rollout restart deployment/welcomebot-master -n welcomebot-local || {
    echo -e "${RED}❌ Failed to restart master deployment${NC}"
    exit 1
}
kubectl rollout restart deployment/welcomebot-worker-slave1 -n welcomebot-local || {
    echo -e "${RED}❌ Failed to restart worker-slave1 deployment${NC}"
    exit 1
}
kubectl rollout restart deployment/welcomebot-worker-slave2 -n welcomebot-local || {
    echo -e "${RED}❌ Failed to restart worker-slave2 deployment${NC}"
    exit 1
}
kubectl rollout restart deployment/welcomebot-worker-slave3 -n welcomebot-local || {
    echo -e "${RED}❌ Failed to restart worker-slave3 deployment${NC}"
    exit 1
}
echo -e "${GREEN}  ✓ Deployments restarted${NC}"
echo ""

echo -e "${BLUE}⏳ Waiting for pods to be ready...${NC}"
kubectl rollout status deployment/welcomebot-master -n welcomebot-local --timeout=60s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Master not ready yet (check logs)${NC}"
}
kubectl rollout status deployment/welcomebot-worker-slave1 -n welcomebot-local --timeout=60s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Worker Slave 1 not ready yet (check logs)${NC}"
}
kubectl rollout status deployment/welcomebot-worker-slave2 -n welcomebot-local --timeout=60s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Worker Slave 2 not ready yet (check logs)${NC}"
}
kubectl rollout status deployment/welcomebot-worker-slave3 -n welcomebot-local --timeout=60s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Worker Slave 3 not ready yet (check logs)${NC}"
}

echo ""
echo -e "${GREEN}✅ Reload complete!${NC}"
echo ""
echo -e "${BLUE}Watch logs with:${NC}"
echo -e "  ${YELLOW}kubectl logs -f deployment/welcomebot-master -n welcomebot-local${NC}"
echo -e "  ${YELLOW}./scripts/dev-logs.sh${NC}"
echo ""
echo -e "${BLUE}Current pod status:${NC}"
kubectl get pods -n welcomebot-local

