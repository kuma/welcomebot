#!/bin/bash
# Production Cleanup Script
# Deletes the entire welcomebot-lightprod namespace

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

NAMESPACE="welcomebot-lightprod"

echo -e "${YELLOW}🧹 Production Cleanup (welcomebot-lightprod)${NC}"
echo ""
echo -e "${RED}⚠️  WARNING: This will DELETE ALL resources in the $NAMESPACE namespace!${NC}"
echo ""
echo "This includes:"
echo "  - All bot pods (master and workers)"
echo "  - PostgreSQL database and ALL DATA"
echo "  - Redis and ALL DATA"
echo "  - All persistent volumes"
echo "  - All secrets and configs"
echo ""

# Show current resources
echo -e "${BLUE}Current resources in $NAMESPACE:${NC}"
kubectl get all -n $NAMESPACE 2>/dev/null || {
    echo -e "${YELLOW}Namespace does not exist or is already empty${NC}"
    exit 0
}
echo ""

# Confirmation
echo -e "${RED}Are you ABSOLUTELY sure you want to delete everything?${NC}"
read -p "Type 'yes' to confirm: " confirmation

if [ "$confirmation" != "yes" ]; then
    echo -e "${GREEN}Cancelled. Nothing was deleted.${NC}"
    exit 0
fi

echo ""
echo -e "${BLUE}🗑️  Deleting namespace $NAMESPACE...${NC}"

# Delete namespace
kubectl delete namespace $NAMESPACE || {
    echo -e "${RED}❌ Failed to delete namespace${NC}"
    echo ""
    echo "If the namespace is stuck in 'Terminating' state, you may need to:"
    echo "  1. Check for finalizers: kubectl get namespace $NAMESPACE -o yaml"
    echo "  2. Manually remove resources with finalizers"
    echo "  3. Force delete PVCs if needed"
    exit 1
}

echo -e "${GREEN}✅ Namespace deleted successfully${NC}"
echo ""
echo -e "${BLUE}💡 To redeploy:${NC}"
echo -e "   ${YELLOW}./scripts/prod-deploy.sh${NC}"
echo ""

