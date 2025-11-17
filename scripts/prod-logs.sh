#!/bin/bash
# Production Logs Viewer
# Interactive script to view logs from production pods

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

NAMESPACE="welcomebot-lightprod"

echo -e "${BLUE}📋 Production Logs Viewer (welcomebot-lightprod)${NC}"
echo ""
echo "Select which logs to view:"
echo ""
echo "  1) Master bot (live - follow)"
echo "  2) Worker slave 1 (live - follow)"
echo "  3) Worker slave 2 (live - follow)"
echo "  4) Worker slave 3 (live - follow)"
echo "  5) Worker slave 4 (live - follow)"
echo "  6) Worker slave 5 (live - follow)"
echo "  7) Worker slave 6 (live - follow)"
echo "  8) Worker slave 7 (live - follow)"
echo "  9) Worker slave 8 (live - follow)"
echo " 10) Worker slave 9 (live - follow)"
echo " 11) Worker slave 10 (live - follow)"
echo " 12) All bot pods (live - follow)"
echo " 13) Master bot (last 100 lines)"
echo " 14) All workers (last 100 lines each)"
echo " 15) PostgreSQL (live - follow)"
echo " 16) PostgreSQL (last 100 lines)"
echo " 17) Redis (live - follow)"
echo " 18) Redis (last 100 lines)"
echo " 19) All pods status"
echo ""
read -p "Enter choice (1-19): " choice

case $choice in
    1)
        echo -e "${GREEN}Following master bot logs...${NC}"
        echo -e "${YELLOW}(Press Ctrl+C to exit)${NC}"
        echo ""
        kubectl logs -f deployment/welcomebot-master -n $NAMESPACE
        ;;
    2|3|4|5|6|7|8|9|10|11)
        slave_num=$((choice - 1))
        echo -e "${GREEN}Following worker slave $slave_num logs...${NC}"
        echo -e "${YELLOW}(Press Ctrl+C to exit)${NC}"
        echo ""
        kubectl logs -f deployment/welcomebot-worker-slave$slave_num -n $NAMESPACE
        ;;
    12)
        echo -e "${GREEN}Following all bot pods logs...${NC}"
        echo -e "${YELLOW}(Press Ctrl+C to exit)${NC}"
        echo ""
        kubectl logs -f -l app=welcomebot -n $NAMESPACE --prefix=true
        ;;
    13)
        echo -e "${GREEN}Master bot (last 100 lines):${NC}"
        echo ""
        kubectl logs deployment/welcomebot-master -n $NAMESPACE --tail=100
        ;;
    14)
        echo -e "${GREEN}All workers (last 100 lines each):${NC}"
        echo ""
        for i in {1..10}; do
            echo -e "${BLUE}=== Worker Slave $i ===${NC}"
            kubectl logs deployment/welcomebot-worker-slave$i -n $NAMESPACE --tail=100
            echo ""
        done
        ;;
    15)
        echo -e "${GREEN}Following PostgreSQL logs...${NC}"
        echo -e "${YELLOW}(Press Ctrl+C to exit)${NC}"
        echo ""
        kubectl logs -f statefulset/postgres -n $NAMESPACE
        ;;
    16)
        echo -e "${GREEN}PostgreSQL (last 100 lines):${NC}"
        echo ""
        kubectl logs statefulset/postgres -n $NAMESPACE --tail=100
        ;;
    17)
        echo -e "${GREEN}Following Redis logs...${NC}"
        echo -e "${YELLOW}(Press Ctrl+C to exit)${NC}"
        echo ""
        kubectl logs -f deployment/redis -n $NAMESPACE
        ;;
    18)
        echo -e "${GREEN}Redis (last 100 lines):${NC}"
        echo ""
        kubectl logs deployment/redis -n $NAMESPACE --tail=100
        ;;
    19)
        echo -e "${GREEN}All pods status:${NC}"
        echo ""
        kubectl get pods -n $NAMESPACE -o wide
        echo ""
        echo -e "${BLUE}Recent events:${NC}"
        kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' | tail -20
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac
