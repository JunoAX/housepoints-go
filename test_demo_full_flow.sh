#!/bin/bash

echo "=== Testing Full API Flow on demo.housepoints.ai ==="
echo ""

# Get parent token
echo "1. Login as parent (demo)"
PARENT_TOKEN=$(curl -s -X POST https://demo.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "demo", "password": "demo"}' | jq -r '.token')

echo "Parent token: ${PARENT_TOKEN:0:50}..."
echo ""

# List users
echo "2. List all users"
curl -s https://demo.housepoints.ai/api/users \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq '.users | length'
echo ""

# Create a chore
echo "3. Create a new chore (parent only)"
NEW_CHORE=$(curl -s -X POST https://demo.housepoints.ai/api/chores \
  -H "Authorization: Bearer $PARENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Chore - API Flow",
    "description": "Created to test full Go backend flow",
    "category": "testing",
    "base_points": 30,
    "difficulty": "medium",
    "requires_verification": true
  }')

CHORE_ID=$(echo "$NEW_CHORE" | jq -r '.id')
echo "Created chore: $CHORE_ID"
echo "$NEW_CHORE" | jq '{id, name, base_points, requires_verification}'
echo ""

# Get child user ID
echo "4. Get child user (mo) ID"
MO_ID=$(curl -s https://demo.housepoints.ai/api/users \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq -r '.users[] | select(.username == "mo") | .id')

echo "Mo's user ID: $MO_ID"
echo ""

# Create assignment
echo "5. Create assignment for mo (parent only)"
NEW_ASSIGNMENT=$(curl -s -X POST https://demo.housepoints.ai/api/assignments \
  -H "Authorization: Bearer $PARENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"chore_id\": \"$CHORE_ID\",
    \"assigned_to\": \"$MO_ID\",
    \"points_offered\": 35,
    \"due_date\": \"2025-10-15\"
  }")

ASSIGNMENT_ID=$(echo "$NEW_ASSIGNMENT" | jq -r '.id')
echo "Created assignment: $ASSIGNMENT_ID"
echo "$NEW_ASSIGNMENT" | jq '{id, status, points_offered, due_date}'
echo ""

# Login as child
echo "6. Login as child (mo)"
# First need to set mo's password
PGPASSWORD='HP_Sec2025_O0mZVY90R1Yg8L' psql -h 10.1.10.20 -U postgres -d family_demo -c \
  "UPDATE users SET password_hash = '\$2a\$10\$DqXiglOXTmc0j7cC3RKS..Lg42g8ncPVphPTobQZU4fLeV5Cxl8M.' WHERE username = 'mo';" > /dev/null 2>&1

CHILD_TOKEN=$(curl -s -X POST https://demo.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "mo", "password": "demo"}' | jq -r '.token')

echo "Child token: ${CHILD_TOKEN:0:50}..."
echo ""

# List child's assignments
echo "7. List mo's assignments"
curl -s "https://demo.housepoints.ai/api/assignments?assigned_to=$MO_ID" \
  -H "Authorization: Bearer $CHILD_TOKEN" | jq '.assignments | length'
echo ""

# Complete assignment
echo "8. Complete assignment (as child)"
curl -s -X POST "https://demo.housepoints.ai/api/assignments/$ASSIGNMENT_ID/complete" \
  -H "Authorization: Bearer $CHILD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"notes": "Completed via Go backend test"}' | jq
echo ""

# Verify assignment
echo "9. Verify assignment (as parent)"
curl -s -X POST "https://demo.housepoints.ai/api/assignments/$ASSIGNMENT_ID/verify" \
  -H "Authorization: Bearer $PARENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"approved": true, "points_awarded": 35, "notes": "Verified via Go backend"}' | jq
echo ""

# Check leaderboard
echo "10. Check weekly leaderboard"
curl -s https://demo.housepoints.ai/api/leaderboard/weekly \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq '.leaderboard[] | select(.username == "mo") | {username, points, weekly_points}'
echo ""

# Cleanup: soft-delete chore
echo "11. Cleanup: Delete test chore"
curl -s -X DELETE "https://demo.housepoints.ai/api/chores/$CHORE_ID" \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq
echo ""

echo "=== Test Complete ==="
