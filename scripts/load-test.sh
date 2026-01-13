#!/usr/bin/env bash
set -euo pipefail

BASE=${BASE:-http://localhost:8080}
TENANT_ID=${TENANT_ID:?TENANT_ID required}
BRANCH_ID=${BRANCH_ID:?BRANCH_ID required}
SERVICE_ID=${SERVICE_ID:?SERVICE_ID required}

if command -v hey >/dev/null 2>&1; then
  hey -n 200 -c 20 -m POST -H "Content-Type: application/json" \
    -d "{\"request_id\":\"$(uuidgen || cat /proc/sys/kernel/random/uuid)\",\"tenant_id\":\"$TENANT_ID\",\"branch_id\":\"$BRANCH_ID\",\"service_id\":\"$SERVICE_ID\"}" \
    "$BASE/api/tickets"
  exit 0
fi

echo "Install 'hey' for load testing: https://github.com/rakyll/hey"
exit 1
