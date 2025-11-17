#!/bin/bash
# Local Development Deployment Script
# Builds images and deploys to local Kubernetes cluster

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting Discord Bot Local Development Deployment${NC}"
echo ""

# Check current kubectl context
CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null || echo "")

if [ -z "$CURRENT_CONTEXT" ]; then
    echo -e "${RED}❌ No Kubernetes cluster detected!${NC}"
    echo ""
    echo "Please enable Kubernetes in Docker Desktop:"
    echo "  Docker Desktop → Settings → Kubernetes → Enable Kubernetes"
    exit 1
fi

# Switch to docker-desktop if not already there
if [ "$CURRENT_CONTEXT" != "docker-desktop" ]; then
    echo -e "${YELLOW}⚠ Current context: $CURRENT_CONTEXT${NC}"
    echo -e "${BLUE}Switching to docker-desktop context...${NC}"
    kubectl config use-context docker-desktop || {
        echo -e "${RED}❌ Failed to switch to docker-desktop context${NC}"
        echo ""
        echo "Please ensure Docker Desktop Kubernetes is enabled:"
        echo "  Docker Desktop → Settings → Kubernetes → Enable Kubernetes"
        exit 1
    }
    echo -e "${GREEN}✓ Switched to docker-desktop${NC}"
else
    echo -e "${BLUE}✓ Using Docker Desktop Kubernetes${NC}"
fi

CLUSTER_TYPE="docker-desktop"
echo ""

# Verify we're in the correct directory
if [ ! -f "go.mod" ] || [ ! -d "cmd/master" ]; then
    echo -e "${RED}❌ Error: Must run from project root directory${NC}"
    exit 1
fi

# Build Docker images
echo -e "${BLUE}📦 Building Docker images...${NC}"
echo "  Building welcomebot-master:local..."
docker build -t welcomebot-master:local . > /dev/null 2>&1 || {
    echo -e "${RED}❌ Failed to build master image${NC}"
    docker build -t welcomebot-master:local .
    exit 1
}
echo -e "${GREEN}  ✓ welcomebot-master:local built${NC}"

echo "  Building welcomebot-worker:local..."
docker build -t welcomebot-worker:local . > /dev/null 2>&1 || {
    echo -e "${RED}❌ Failed to build worker image${NC}"
    docker build -t welcomebot-worker:local .
    exit 1
}
echo -e "${GREEN}  ✓ welcomebot-worker:local built${NC}"
echo ""

# Docker Desktop uses local Docker daemon - no need to load images
echo -e "${BLUE}📥 Images ready (Docker Desktop uses local Docker daemon)${NC}"
echo -e "${GREEN}  ✓ welcomebot-master:local available${NC}"
echo -e "${GREEN}  ✓ welcomebot-worker:local available${NC}"
echo ""

# Check if secrets.env exists
SECRETS_FILE="deployments/overlays/local/secrets.env"
if [ ! -f "$SECRETS_FILE" ]; then
    echo -e "${YELLOW}⚠ Warning: $SECRETS_FILE not found${NC}"
    echo ""
    echo "Creating example secrets file..."
    cat > "$SECRETS_FILE" << 'EOF'
# Welcome Bot - Local Development Secrets
# Copy this file and fill in your values

# Master Bot Token
DISCORD_BOT_TOKEN=your_master_bot_token_here

# Slave Bot Tokens (3 separate bot accounts)
SLAVE_1_TOKEN=your_slave1_bot_token_here
SLAVE_2_TOKEN=your_slave2_bot_token_here
SLAVE_3_TOKEN=your_slave3_bot_token_here

# PostgreSQL Configuration
POSTGRES_USER=welcomebot_dev
POSTGRES_PASSWORD=dev_password
POSTGRES_DB=welcomebot_dev

# Redis Password (optional for local)
REDIS_PASSWORD=
EOF
    echo -e "${YELLOW}  ✓ Created $SECRETS_FILE${NC}"
    echo ""
    echo -e "${RED}Please edit $SECRETS_FILE and add your 4 Discord bot tokens!${NC}"
    echo ""
    echo "You need tokens for:"
    echo "  1. Master bot (DISCORD_BOT_TOKEN)"
    echo "  2. Slave 1 bot (SLAVE_1_TOKEN)"
    echo "  3. Slave 2 bot (SLAVE_2_TOKEN)"
    echo "  4. Slave 3 bot (SLAVE_3_TOKEN)"
    echo ""
    echo "See TOKEN_SETUP_GUIDE.md for how to create bot accounts."
    echo "Then run this script again."
    exit 1
fi

# Apply Kubernetes manifests
echo -e "${BLUE}☸️  Applying Kubernetes manifests...${NC}"
kubectl apply -k deployments/overlays/local || {
    echo -e "${RED}❌ Failed to apply manifests${NC}"
    exit 1
}
echo -e "${GREEN}  ✓ Manifests applied${NC}"
echo ""

# Wait for pods to be ready
echo -e "${BLUE}⏳ Waiting for pods to be ready...${NC}"
echo "  (this may take 1-2 minutes)"
echo ""

# Wait for postgres
kubectl wait --for=condition=ready pod -l app=postgres -n welcomebot-local --timeout=120s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ PostgreSQL pod not ready yet (continuing anyway)${NC}"
}

# Wait for redis
kubectl wait --for=condition=ready pod -l app=redis -n welcomebot-local --timeout=120s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Redis pod not ready yet (continuing anyway)${NC}"
}

# Wait for master
kubectl wait --for=condition=ready pod -l component=master -n welcomebot-local --timeout=120s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Master pod not ready yet (check logs)${NC}"
}

# Wait for workers (3 slaves)
kubectl wait --for=condition=ready pod -l slave=slave-1 -n welcomebot-local --timeout=120s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Worker slave-1 pod not ready yet (check logs)${NC}"
}
kubectl wait --for=condition=ready pod -l slave=slave-2 -n welcomebot-local --timeout=120s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Worker slave-2 pod not ready yet (check logs)${NC}"
}
kubectl wait --for=condition=ready pod -l slave=slave-3 -n welcomebot-local --timeout=120s 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ Worker slave-3 pod not ready yet (check logs)${NC}"
}

echo ""
echo -e "${GREEN}✅ Deployment complete!${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}📊 Quick Commands:${NC}"
echo ""
echo "  View all pods:"
echo -e "    ${YELLOW}kubectl get pods -n welcomebot-local${NC}"
echo ""
echo "  Watch pod status:"
echo -e "    ${YELLOW}kubectl get pods -n welcomebot-local -w${NC}"
echo ""
echo "  Master bot logs:"
echo -e "    ${YELLOW}kubectl logs -f deployment/welcomebot-master -n welcomebot-local${NC}"
echo ""
echo "  Worker bot logs (slave 1):"
echo -e "    ${YELLOW}kubectl logs -f deployment/welcomebot-worker-slave1 -n welcomebot-local${NC}"
echo ""
echo "  Worker bot logs (slave 2):"
echo -e "    ${YELLOW}kubectl logs -f deployment/welcomebot-worker-slave2 -n welcomebot-local${NC}"
echo ""
echo "  Worker bot logs (slave 3):"
echo -e "    ${YELLOW}kubectl logs -f deployment/welcomebot-worker-slave3 -n welcomebot-local${NC}"
echo ""
echo "  Shell into master:"
echo -e "    ${YELLOW}kubectl exec -it deployment/welcomebot-master -n welcomebot-local -- sh${NC}"
echo ""
echo "  Reload after code changes:"
echo -e "    ${YELLOW}./scripts/dev-reload.sh${NC}"
echo ""
echo "  View logs (interactive):"
echo -e "    ${YELLOW}./scripts/dev-logs.sh${NC}"
echo ""
echo "  Clean up everything:"
echo -e "    ${YELLOW}./scripts/dev-clean.sh${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Show current pod status
echo -e "${BLUE}Current status:${NC}"
kubectl get pods -n welcomebot-local

