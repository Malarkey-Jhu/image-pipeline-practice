#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Usage: $0 <media_id>"
  exit 1
fi

MEDIA_ID="$1"
NETWORK="${SYS_DESIGN_NETWORK:-sys-design_default}"

# 1) Get task_id and output_key
TASK_INFO=$(docker compose exec -T postgres psql -U app -d app -c "select id, output_key from processing_task where media_id='${MEDIA_ID}' order by created_at desc limit 1;")
TASK_ID=$(echo "$TASK_INFO" | awk 'NR==3 {print $1}')
OUTPUT_KEY=$(echo "$TASK_INFO" | awk 'NR==3 {print $3}')

if [ -z "$TASK_ID" ] || [ -z "$OUTPUT_KEY" ]; then
  echo "No task found for media_id: $MEDIA_ID"
  exit 1
fi

# 2) Create fake output

docker run --rm --network "$NETWORK" \
  -e MC_HOST_local="http://minioadmin:minioadmin@minio:9000" \
  minio/mc:latest \
  pipe "local/media/${OUTPUT_KEY}" < /dev/null

# 3) Publish message with same task_id
PUBLISH_BODY=$(python3 - <<PY
import json
task = {"task_id": "${TASK_ID}", "media_id": "${MEDIA_ID}", "step": "resize"}
body = {
  "properties": {},
  "routing_key": "processing_tasks",
  "payload": json.dumps(task),
  "payload_encoding": "string",
}
print(json.dumps(body))
PY
)

curl -s -u guest:guest -H "Content-Type: application/json" \
  -X POST http://localhost:15672/api/exchanges/%2F/amq.default/publish \
  -d "$PUBLISH_BODY"

sleep 1

echo "Published task_id: $TASK_ID"
echo "Checking worker logs (last 50 lines)..."
docker compose logs --tail=50 worker
