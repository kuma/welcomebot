#!/bin/bash
# Production Shell Access
# Interactive script to get shell access to production pods

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

NAMESPACE="welcomebot-lightprod"

echo -e "${BLUE}🐚 Production Shell Access (welcomebot-lightprod)${NC}"
echo ""
echo "Select which pod to access:"
echo ""
echo "  1) Master bot"
echo "  2) Worker slave 1"
echo "  3) Worker slave 2"
echo "  4) Worker slave 3"
echo "  5) Worker slave 4"
echo "  6) Worker slave 5"
echo "  7) Worker slave 6"
echo "  8) Worker slave 7"
echo "  9) Worker slave 8"
echo " 10) Worker slave 9"
echo " 11) Worker slave 10"
echo " 12) PostgreSQL"
echo " 13) Redis"
echo ""
read -p "Enter choice (1-13): " choice

case $choice in
    1)
        echo -e "${GREEN}Opening shell in master bot...${NC}"
        echo -e "${YELLOW}(Type 'exit' to return)${NC}"
        echo ""
        kubectl exec -it deployment/welcomebot-master -n $NAMESPACE -- sh
        ;;
    2|3|4|5|6|7|8|9|10|11)
        slave_num=$((choice - 1))
        echo -e "${GREEN}Opening shell in worker slave $slave_num...${NC}"
        echo -e "${YELLOW}(Type 'exit' to return)${NC}"
        echo ""
        kubectl exec -it deployment/welcomebot-worker-slave$slave_num -n $NAMESPACE -- sh
        ;;
    12)
        echo -e "${GREEN}Opening shell in PostgreSQL...${NC}"
        echo -e "${YELLOW}(Type 'exit' to return)${NC}"
        echo ""
        echo -e "${BLUE}💡 Tip: Connect to database with:${NC}"
        echo -e "   ${YELLOW}psql -U \$POSTGRES_USER -d \$POSTGRES_DB${NC}"
        echo ""
        kubectl exec -it statefulset/postgres -n $NAMESPACE -- sh
        ;;
    13)
        echo -e "${GREEN}Opening shell in Redis...${NC}"
        echo -e "${YELLOW}(Type 'exit' to return)${NC}"
        echo ""
        echo -e "${BLUE}💡 Tip: Connect to Redis with:${NC}"
        echo -e "   ${YELLOW}redis-cli${NC}"
        echo ""
        kubectl exec -it deployment/redis -n $NAMESPACE -- sh
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac
