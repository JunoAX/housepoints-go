#!/bin/bash

echo "=== Testing Gamull Family Migration on gamull.housepoints.ai ==="
echo ""

# Get parent token
echo "1. Login as parent (tom)"
PARENT_TOKEN=$(curl -s -X POST https://gamull.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "tom", "password": "tom"}' | jq -r '.token')

echo "Parent token: ${PARENT_TOKEN:0:50}..."
echo ""

# List users
echo "2. List all users"
USERS_COUNT=$(curl -s https://gamull.housepoints.ai/api/users \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq '.users | length')
echo "Total users: $USERS_COUNT"
echo ""

# List chores
echo "3. List existing chores"
CHORES_COUNT=$(curl -s https://gamull.housepoints.ai/api/chores \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq '.chores | length')
echo "Total chores: $CHORES_COUNT"
curl -s https://gamull.housepoints.ai/api/chores \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq '.chores[0:3] | map({name, base_points, category})'
echo ""

# List assignments
echo "4. List existing assignments"
ASSIGNMENTS=$(curl -s https://gamull.housepoints.ai/api/assignments \
  -H "Authorization: Bearer $PARENT_TOKEN")
ASSIGNMENTS_COUNT=$(echo "$ASSIGNMENTS" | jq '.assignments | length')
echo "Total assignments: $ASSIGNMENTS_COUNT"
echo "$ASSIGNMENTS" | jq '.assignments[0:3] | map({chore_name, status, points_offered, assigned_to_name})'
echo ""

# Check leaderboard
echo "5. Check weekly leaderboard (verify points)"
curl -s https://gamull.housepoints.ai/api/leaderboard/weekly \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq '.leaderboard | map({username, points, weekly_points, total_points})'
echo ""

# Login as child
echo "6. Login as child (mo)"
CHILD_TOKEN=$(curl -s -X POST https://gamull.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "mo", "password": "mo"}' | jq -r '.token')

echo "Child token: ${CHILD_TOKEN:0:50}..."
echo ""

# Get Mo's user ID
echo "7. Get mo's profile"
MO_PROFILE=$(curl -s https://gamull.housepoints.ai/api/users/me \
  -H "Authorization: Bearer $CHILD_TOKEN")
echo "$MO_PROFILE" | jq '{username, display_name, total_points, weekly_points, level}'
MO_ID=$(echo "$MO_PROFILE" | jq -r '.id')
echo ""

# List mo's assignments
echo "8. List mo's assignments"
curl -s "https://gamull.housepoints.ai/api/assignments?assigned_to=$MO_ID" \
  -H "Authorization: Bearer $CHILD_TOKEN" | jq '.assignments | length'
echo ""

# Get family schedule
echo "9. Get family schedule"
curl -s "https://gamull.housepoints.ai/api/schedule?start_date=2025-10-06&end_date=2025-10-12" \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq '.schedule | length'
echo ""

echo "=== Migration Test Complete ==="
