#!/bin/bash
# Clean Up Local Development Environment
# Deletes the local namespace and resources

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🧹 Cleaning Up Discord Bot Local Development${NC}"
echo ""

# Check if namespace exists
if ! kubectl get namespace welcomebot-local &> /dev/null; then
    echo -e "${YELLOW}⚠ Namespace 'welcomebot-local' not found${NC}"
    echo "Nothing to clean up!"
    exit 0
fi

# Show what will be deleted
echo -e "${BLUE}Current resources in welcomebot-local:${NC}"
kubectl get all -n welcomebot-local 2>/dev/null || true
echo ""

# Confirmation
echo -e "${RED}⚠️  This will DELETE all resources in the 'welcomebot-local' namespace!${NC}"
echo ""
read -p "Are you sure? (y/N): " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled."
    exit 0
fi

echo ""
echo -e "${BLUE}🗑️  Deleting namespace...${NC}"
kubectl delete namespace welcomebot-local

echo ""
echo -e "${GREEN}✅ Cleanup complete!${NC}"
echo ""
echo -e "${BLUE}To redeploy:${NC}"
echo -e "  ${YELLOW}./scripts/dev-local.sh${NC}"

