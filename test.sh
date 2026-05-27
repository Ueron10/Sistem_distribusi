#!/bin/bash

# Test script for Distributed Counter System
# This script demonstrates basic functionality of the system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Distributed Counter System Test Script${NC}"
echo "=================================="
echo ""

# Base URLs for the three nodes
NODE1="http://localhost:8080"
NODE2="http://localhost:8081"
NODE3="http://localhost:8082"

# Function to check if a node is healthy
check_health() {
    local node=$1
    local response=$(curl -s "$node/health")
    if echo "$response" | grep -q "healthy"; then
        echo -e "${GREEN}✓${NC} Node at $node is healthy"
        return 0
    else
        echo -e "${RED}✗${NC} Node at $node is not responding"
        return 1
    fi
}

# Function to increment counter
increment_counter() {
    local node=$1
    local delta=$2
    echo "Incrementing counter by $delta at $node"
    curl -s -X POST "$node/counter/increment" \
        -H "Content-Type: application/json" \
        -d "{\"delta\": $delta}" | jq '.'
    echo ""
}

# Function to get counter value
get_value() {
    local node=$1
    echo "Getting counter value from $node"
    curl -s "$node/counter/value" | jq '.'
    echo ""
}

# Function to list peers
list_peers() {
    local node=$1
    echo "Peers at $node:"
    curl -s "$node/peers" | jq '.'
    echo ""
}

# Check if nodes are running
echo -e "${YELLOW}Checking node health...${NC}"
check_health $NODE1
check_health $NODE2
check_health $NODE3
echo ""

# Test 1: Increment on different nodes
echo -e "${YELLOW}Test 1: Increment counter on different nodes${NC}"
increment_counter $NODE1 5
increment_counter $NODE2 3
increment_counter $NODE3 2
echo ""

# Wait for gossip to propagate
echo -e "${YELLOW}Waiting for gossip propagation (5 seconds)...${NC}"
sleep 5
echo ""

# Test 2: Check values on all nodes
echo -e "${YELLOW}Test 2: Check counter values on all nodes${NC}"
get_value $NODE1
get_value $NODE2
get_value $NODE3
echo ""

# Test 3: Add a new peer dynamically
echo -e "${YELLOW}Test 3: List peers${NC}"
list_peers $NODE1
echo ""

# Test 4: Increment multiple times
echo -e "${YELLOW}Test 4: Increment multiple times on single node${NC}"
for i in {1..3}; do
    increment_counter $NODE1 1
    sleep 1
done
echo ""

# Wait for gossip
echo -e "${YELLOW}Waiting for gossip propagation (5 seconds)...${NC}"
sleep 5
echo ""

# Final value check
echo -e "${YELLOW}Test 5: Final counter values${NC}"
get_value $NODE1
get_value $NODE2
get_value $NODE3
echo ""

echo -e "${GREEN}Test completed!${NC}"
