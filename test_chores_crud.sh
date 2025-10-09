#!/bin/bash

# Get parent token (tom is a parent)
PARENT_TOKEN=$(curl -s -X POST https://gamull.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "tom", "password": "tom"}' | jq -r '.token')

# Get child token (mo is a child)
CHILD_TOKEN=$(curl -s -X POST https://gamull.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "mo", "password": "mo"}' | jq -r '.token')

echo "=== Testing Chores CRUD Endpoints ==="
echo ""

echo "1. Test POST /api/chores (create chore as parent)"
NEW_CHORE=$(curl -s -X POST https://gamull.housepoints.ai/api/chores \
  -H "Authorization: Bearer $PARENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Chore from Go API",
    "description": "Created via automated test",
    "category": "testing",
    "base_points": 25,
    "difficulty": "easy",
    "requires_verification": false
  }' | jq)

echo "$NEW_CHORE"
CHORE_ID=$(echo "$NEW_CHORE" | jq -r '.id')

echo ""
echo "2. Test POST /api/chores as child (should fail)"
curl -s -X POST https://gamull.housepoints.ai/api/chores \
  -H "Authorization: Bearer $CHILD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Should Fail", "category": "testing"}' | jq

echo ""
echo "3. Test PUT /api/chores/:id (update chore as parent)"
curl -s -X PUT "https://gamull.housepoints.ai/api/chores/$CHORE_ID" \
  -H "Authorization: Bearer $PARENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "base_points": 50,
    "description": "Updated via automated test"
  }' | jq

echo ""
echo "4. Test DELETE /api/chores/:id (soft delete as parent)"
curl -s -X DELETE "https://gamull.housepoints.ai/api/chores/$CHORE_ID" \
  -H "Authorization: Bearer $PARENT_TOKEN" | jq

echo ""
echo "5. Verify chore is soft-deleted (active=false)"
echo "Created chore ID was: $CHORE_ID"
