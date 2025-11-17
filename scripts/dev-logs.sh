#!/bin/bash
# View Logs from Local Development Pods
# Shows logs from master and worker

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
echo -e "${BLUE}📋 Discord Bot Local Development - Logs${NC}"
echo ""
echo "Select which logs to view:"
echo ""
echo "  1) Master bot (live)"
echo "  2) Worker bot (live)"
echo "  3) All pods (live)"
echo "  4) Master bot (last 100 lines)"
echo "  5) Worker bot (last 100 lines)"
echo "  6) PostgreSQL"
echo "  7) Redis"
echo ""
read -p "Choice (1-7): " -n 1 -r
echo ""
echo ""

case $REPLY in
    1)
        echo -e "${BLUE}Following master bot logs (Ctrl+C to exit)...${NC}"
        echo ""
        kubectl logs -f deployment/welcomebot-master -n welcomebot-local
        ;;
    2)
        echo -e "${BLUE}Following worker bot logs (Ctrl+C to exit)...${NC}"
        echo ""
        kubectl logs -f deployment/welcomebot-worker -n welcomebot-local
        ;;
    3)
        echo -e "${BLUE}Following all pod logs (Ctrl+C to exit)...${NC}"
        echo ""
        kubectl logs -f -l app=welcomebot -n welcomebot-local --all-containers=true --prefix=true
        ;;
    4)
        echo -e "${BLUE}Master bot - last 100 lines:${NC}"
        echo ""
        kubectl logs deployment/welcomebot-master -n welcomebot-local --tail=100
        ;;
    5)
        echo -e "${BLUE}Worker bot - last 100 lines:${NC}"
        echo ""
        kubectl logs deployment/welcomebot-worker -n welcomebot-local --tail=100
        ;;
    6)
        echo -e "${BLUE}PostgreSQL logs:${NC}"
        echo ""
        kubectl logs -f -l app=postgres -n welcomebot-local
        ;;
    7)
        echo -e "${BLUE}Redis logs:${NC}"
        echo ""
        kubectl logs -f -l app=redis -n welcomebot-local
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

