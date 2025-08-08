#!/bin/bash

echo "Testing eBPF Game API Server"
echo "=============================="

# Base URL
BASE_URL="http://localhost:8080"

echo ""
echo "1. Getting available APIs..."
curl -s $BASE_URL/apis | jq .

echo ""
echo "2. Adding IPs..."
curl -s -X POST $BASE_URL/add_ips \
  -H "Content-Type: application/json" \
  -d '{"add_ips": [1,2,3]}' | jq .

echo ""
echo "3. Adding more IPs..."
curl -s -X POST $BASE_URL/add_ips \
  -H "Content-Type: application/json" \
  -d '{"add_ips": [4,5,6]}' | jq .

echo ""
echo "4. Printing all IPs..."
curl -s -X POST $BASE_URL/print_all_ips \
  -H "Content-Type: application/json" \
  -d '{"print_all_ips": true}' | jq .

echo ""
echo "5. Clearing IP list..."
curl -s -X POST $BASE_URL/clear_ip_list \
  -H "Content-Type: application/json" \
  -d '{"clear_ip_list": true}' | jq .

echo ""
echo "6. Printing all IPs (should be empty)..."
curl -s -X POST $BASE_URL/print_all_ips \
  -H "Content-Type: application/json" \
  -d '{"print_all_ips": true}' | jq .

echo ""
echo "API testing complete!" 