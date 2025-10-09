#!/bin/bash

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiOTkxMTI2NGYtYzY2Yy00MjQwLWFhMDEtY2MxMWI5Njk0NzgxIiwiZmFtaWx5X2lkIjoiM2YzMmYyNTctYWQ0Mi00M2Q2LWFlMGEtOGU0ZGI5YzFjZTU1IiwidXNlcm5hbWUiOiJtbyIsImlzX3BhcmVudCI6ZmFsc2UsImlzcyI6ImhvdXNlcG9pbnRzLWdvIiwiZXhwIjoxNzYwMDM4NTA4LCJuYmYiOjE3NTk5NTIxMDgsImlhdCI6MTc1OTk1MjEwOH0.nzXB-yZtCxpblLFuQkxIchUaCYGCpoP7TKZ3H0YLYYw"

curl -s -X POST https://gamull.housepoints.ai/api/rewards/b92d0a07-9bb3-45fb-b082-51ae30b20a2c/redeem \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"notes": "Testing redemption from Go backend"}' | jq
