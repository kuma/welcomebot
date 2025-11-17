#!/bin/bash
# Production Deployment Script
# Builds images, pushes to Harbor, and deploys to production Kubernetes cluster

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting Production Deployment to welcomebot-lightprod${NC}"
echo ""

# Configuration - Harbor settings
HARBOR_REGISTRY="${HARBOR_REGISTRY:-harbor.morikuma.org}"
HARBOR_PROJECT="${HARBOR_PROJECT:-welcomebot}"
HARBOR_USERNAME="${HARBOR_USERNAME:-admin}"
HARBOR_PASSWORD="${HARBOR_PASSWORD:-!Containerisno1}"
IMAGE_TAG="${IMAGE_TAG:-lightprod}"

echo -e "${BLUE}Configuration:${NC}"
echo "  Registry: $HARBOR_REGISTRY"
echo "  Project: $HARBOR_PROJECT"
echo "  Username: $HARBOR_USERNAME"
echo "  Tag: $IMAGE_TAG"
echo ""

# Verify we're in the correct directory
if [ ! -f "go.mod" ] || [ ! -d "cmd/master" ]; then
    echo -e "${RED}❌ Error: Must run from project root directory${NC}"
    exit 1
fi

# Login to Harbor
echo -e "${BLUE}🔐 Logging in to Harbor registry...${NC}"
echo "$HARBOR_PASSWORD" | docker login "$HARBOR_REGISTRY" -u "$HARBOR_USERNAME" --password-stdin || {
    echo -e "${RED}❌ Failed to login to Harbor${NC}"
    exit 1
}
echo -e "${GREEN}  ✓ Logged in to Harbor${NC}"
echo ""

# Setup Docker buildx for multi-architecture builds
echo -e "${BLUE}🔧 Setting up Docker buildx...${NC}"
docker buildx inspect multiarch-builder > /dev/null 2>&1 || {
    docker buildx create --name multiarch-builder --use || {
        echo -e "${RED}❌ Failed to create buildx builder${NC}"
        exit 1
    }
}
docker buildx use multiarch-builder > /dev/null 2>&1
echo -e "${GREEN}  ✓ Buildx ready${NC}"
echo ""

# Build and push multi-architecture images (amd64 + arm64)
echo -e "${BLUE}📦 Building multi-arch images (amd64 + arm64)...${NC}"
echo "  Building and pushing images (contains both master and worker)..."
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -t "$HARBOR_REGISTRY/$HARBOR_PROJECT/welcomebot-master:$IMAGE_TAG" \
    -t "$HARBOR_REGISTRY/$HARBOR_PROJECT/welcomebot-worker:$IMAGE_TAG" \
    --push \
    . || {
    echo -e "${RED}❌ Failed to build/push images${NC}"
    exit 1
}
echo -e "${GREEN}  ✓ Images built and pushed (master and worker)${NC}"
echo ""

# Check if secrets.env exists
SECRETS_FILE="deployments/overlays/lightprod/secrets.env"
if [ ! -f "$SECRETS_FILE" ]; then
    echo -e "${YELLOW}⚠ Warning: $SECRETS_FILE not found${NC}"
    echo ""
    echo "Please create secrets file from example:"
    echo "  cp deployments/overlays/lightprod/secrets.env.example $SECRETS_FILE"
    echo "  vim $SECRETS_FILE  # Edit with your production values"
    echo ""
    echo "You need to configure:"
    echo "  1. Master bot token (DISCORD_BOT_TOKEN)"
    echo "  2. Slave bot tokens (SLAVE_1_TOKEN through SLAVE_10_TOKEN)"
    echo "  3. PostgreSQL credentials"
    echo "  4. Redis password"
    echo ""
    read -p "Continue without secrets? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if Harbor registry secret exists
echo -e "${BLUE}🔑 Checking Harbor registry secret...${NC}"
kubectl get secret harbor-registry -n welcomebot-lightprod > /dev/null 2>&1 || {
    echo -e "${YELLOW}  ⚠ Harbor registry secret not found${NC}"
    echo ""
    echo "Creating Harbor registry secret..."
    
    kubectl create secret docker-registry harbor-registry \
        --docker-server="$HARBOR_REGISTRY" \
        --docker-username="$HARBOR_USERNAME" \
        --docker-password="$HARBOR_PASSWORD" \
        -n welcomebot-lightprod || {
        echo -e "${RED}❌ Failed to create registry secret${NC}"
        exit 1
    }
    echo -e "${GREEN}  ✓ Registry secret created${NC}"
}
echo -e "${GREEN}  ✓ Harbor registry secret exists${NC}"
echo ""

# Apply Kubernetes manifests
echo -e "${BLUE}☸️  Applying Kubernetes manifests...${NC}"
kubectl apply -k deployments/overlays/lightprod || {
    echo -e "${RED}❌ Failed to apply manifests${NC}"
    exit 1
}
echo -e "${GREEN}  ✓ Manifests applied${NC}"
echo ""

# Wait for pods to be ready
echo -e "${BLUE}⏳ Waiting for pods to be ready...${NC}"
echo "  (this may take 2-3 minutes for first deployment)"
echo ""

# Wait for postgres
kubectl wait --for=condition=ready pod -l app=postgres -n welcomebot-lightprod --timeout=180s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ PostgreSQL pod not ready yet (check logs if needed)${NC}"
}

# Wait for redis
kubectl wait --for=condition=ready pod -l app=redis -n welcomebot-lightprod --timeout=180s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Redis pod not ready yet (check logs if needed)${NC}"
}

# Wait for master
kubectl wait --for=condition=ready pod -l component=master -n welcomebot-lightprod --timeout=180s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Master pod not ready yet (check logs)${NC}"
}

# Wait for workers (10 slaves)
for i in {1..10}; do
    kubectl wait --for=condition=ready pod -l slave=slave-$i -n welcomebot-lightprod --timeout=180s 2>/dev/null || {
        echo -e "${YELLOW}  ⚠ Worker slave-$i pod not ready yet (check logs)${NC}"
    }
done

echo ""
echo -e "${GREEN}✅ Production deployment complete!${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}📊 Quick Commands:${NC}"
echo ""
echo "  View all pods:"
echo -e "    ${YELLOW}kubectl get pods -n welcomebot-lightprod${NC}"
echo ""
echo "  View logs:"
echo -e "    ${YELLOW}./scripts/prod-logs.sh${NC}"
echo ""
echo "  Shell into pods:"
echo -e "    ${YELLOW}./scripts/prod-shell.sh${NC}"
echo ""
echo "  Reload after code changes:"
echo -e "    ${YELLOW}./scripts/prod-reload.sh${NC}"
echo ""
echo "  Clean up everything:"
echo -e "    ${YELLOW}./scripts/prod-clean.sh${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Show current pod status
echo -e "${BLUE}Current status:${NC}"
kubectl get pods -n welcomebot-lightprod

