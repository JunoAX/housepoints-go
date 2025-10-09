#!/bin/bash

# Login as Mo to get a Go backend token
curl -s -X POST https://gamull.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "mo", "password": "mo"}' | jq -r '.token'
