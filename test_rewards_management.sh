#!/bin/bash

echo "=== Testing Rewards Management on gamull.housepoints.ai ==="
echo ""

# Login as parent
echo "1. Login as parent (tom)"
TOKEN=$(curl -s -X POST https://gamull.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "tom", "password": "tom"}' | jq -r '.token')

echo "Token: ${TOKEN:0:50}..."
echo ""

# Create a reward
echo "2. Create a new reward"
NEW_REWARD=$(curl -s -X POST https://gamull.housepoints.ai/api/rewards \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Reward - Go API",
    "description": "Testing rewards management in Go backend",
    "cost_points": 50,
    "category": "item",
    "icon": "🎁",
    "active": true,
    "requires_parent_approval": false
  }')

REWARD_ID=$(echo "$NEW_REWARD" | jq -r '.id')
echo "Created reward: $REWARD_ID"
echo "$NEW_REWARD" | jq
echo ""

# Update the reward
echo "3. Update the reward"
curl -s -X PUT "https://gamull.housepoints.ai/api/rewards/$REWARD_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "cost_points": 75,
    "description": "Updated description via Go API"
  }' | jq
echo ""

# List rewards
echo "4. List all rewards"
curl -s https://gamull.housepoints.ai/api/rewards \
  -H "Authorization: Bearer $TOKEN" | jq '.rewards | length'
echo ""

# Delete the reward
echo "5. Delete the test reward"
curl -s -X DELETE "https://gamull.housepoints.ai/api/rewards/$REWARD_ID" \
  -H "Authorization: Bearer $TOKEN" | jq
echo ""

echo "=== Test Complete ==="
