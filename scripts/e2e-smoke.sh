#!/usr/bin/env bash
set -euo pipefail

BASE=${BASE:-http://localhost:8080}
TENANT_ID=${TENANT_ID:?TENANT_ID required}
BRANCH_ID=${BRANCH_ID:?BRANCH_ID required}
SERVICE_ID=${SERVICE_ID:?SERVICE_ID required}
COUNTER_ID=${COUNTER_ID:?COUNTER_ID required}

REQUEST_ID=$(uuidgen || cat /proc/sys/kernel/random/uuid)

curl -sS -X POST "$BASE/api/tickets" \
  -H "Content-Type: application/json" \
  -d "{\"request_id\":\"$REQUEST_ID\",\"tenant_id\":\"$TENANT_ID\",\"branch_id\":\"$BRANCH_ID\",\"service_id\":\"$SERVICE_ID\"}" >/tmp/qms_ticket.json

echo "Ticket created: $(cat /tmp/qms_ticket.json)"

REQUEST_ID=$(uuidgen || cat /proc/sys/kernel/random/uuid)

curl -sS -X POST "$BASE/api/tickets/actions/call-next" \
  -H "Content-Type: application/json" \
  -d "{\"request_id\":\"$REQUEST_ID\",\"tenant_id\":\"$TENANT_ID\",\"branch_id\":\"$BRANCH_ID\",\"service_id\":\"$SERVICE_ID\",\"counter_id\":\"$COUNTER_ID\"}" >/tmp/qms_call.json

echo "Call next result: $(cat /tmp/qms_call.json)"
