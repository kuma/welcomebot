#!/bin/bash
# Shell into Local Development Pods
# Opens an interactive shell in a pod

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if namespace exists
if ! kubectl get namespace welcomebot-local &> /dev/null; then
    echo -e "${RED}❌ Namespace 'welcomebot-local' not found${NC}"
    echo "Run: ./scripts/dev-local.sh first"
    exit 1
fi

# Show menu
echo -e "${BLUE}🐚 Discord Bot Local Development - Shell${NC}"
echo ""
echo "Select which pod to shell into:"
echo ""
echo "  1) Master bot"
echo "  2) Worker bot"
echo "  3) PostgreSQL"
echo "  4) Redis"
echo ""
read -p "Choice (1-4): " -n 1 -r
echo ""
echo ""

case $REPLY in
    1)
        echo -e "${BLUE}Opening shell in master bot...${NC}"
        echo ""
        kubectl exec -it deployment/welcomebot-master -n welcomebot-local -- sh
        ;;
    2)
        echo -e "${BLUE}Opening shell in worker bot...${NC}"
        echo ""
        kubectl exec -it deployment/welcomebot-worker -n welcomebot-local -- sh
        ;;
    3)
        echo -e "${BLUE}Opening shell in PostgreSQL...${NC}"
        echo ""
        kubectl exec -it -l app=postgres -n welcomebot-local -- sh
        ;;
    4)
        echo -e "${BLUE}Opening shell in Redis...${NC}"
        echo ""
        kubectl exec -it -l app=redis -n welcomebot-local -- sh
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

