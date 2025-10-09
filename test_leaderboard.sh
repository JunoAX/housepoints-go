#!/bin/bash

# Get fresh token
TOKEN=$(curl -s -X POST https://gamull.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "mo", "password": "mo"}' | jq -r '.token')

echo "=== Testing Leaderboard Endpoints ==="
echo ""

echo "1. Test GET /api/leaderboard/weekly (weekly points ranking)"
curl -s https://gamull.housepoints.ai/api/leaderboard/weekly \
  -H "Authorization: Bearer $TOKEN" | jq

echo ""
echo "2. Test GET /api/leaderboard/alltime (all-time points ranking)"
curl -s https://gamull.housepoints.ai/api/leaderboard/alltime \
  -H "Authorization: Bearer $TOKEN" | jq
