#!/bin/bash

# Get fresh token
TOKEN=$(curl -s -X POST https://gamull.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "mo", "password": "mo"}' | jq -r '.token')

echo "=== Testing Family Schedule Endpoint ==="
echo ""

echo "1. Test GET /api/schedule (default 30 days from today)"
curl -s "https://gamull.housepoints.ai/api/schedule" \
  -H "Authorization: Bearer $TOKEN" | jq

echo ""
echo "2. Test GET /api/schedule?days=7 (next 7 days)"
curl -s "https://gamull.housepoints.ai/api/schedule?days=7" \
  -H "Authorization: Bearer $TOKEN" | jq

echo ""
echo "3. Test GET /api/schedule with specific date range"
curl -s "https://gamull.housepoints.ai/api/schedule?start_date=2025-10-05&end_date=2025-10-12" \
  -H "Authorization: Bearer $TOKEN" | jq
