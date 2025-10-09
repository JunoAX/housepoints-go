#!/bin/bash

# Get fresh token
TOKEN=$(curl -s -X POST https://gamull.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "mo", "password": "mo"}' | jq -r '.token')

echo "=== Testing User Settings Endpoints ==="
echo ""

echo "1. Test PATCH /api/users/me (update auto_approve_work toggle)"
curl -s -X PATCH https://gamull.housepoints.ai/api/users/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"auto_approve_work": true, "daily_goal": 150}' | jq

echo ""
echo "2. Test PUT /api/users/me/preferences (update notification preferences)"
curl -s -X PUT https://gamull.housepoints.ai/api/users/me/preferences \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"preferences": {"notifications": {"chore_reminders": false, "email_enabled": true}, "general": {"theme": "dark"}}}' | jq

echo ""
echo "3. Test GET /api/users/me (verify changes)"
curl -s https://gamull.housepoints.ai/api/users/me \
  -H "Authorization: Bearer $TOKEN" | jq '{auto_approve_work, daily_goal, preferences}'
