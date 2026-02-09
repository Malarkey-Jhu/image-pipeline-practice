#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Usage: $0 /path/to/image.jpg"
  exit 1
fi

FILE="$1"
if [ ! -f "$FILE" ]; then
  echo "File not found: $FILE"
  exit 1
fi

RESP=$(curl -s -X POST http://localhost:8080/upload-url \
  -H "Content-Type: application/json" \
  -d '{"content_type":"image/jpeg","file_name":"test.jpg"}')

UPLOAD_URL=$(python3 -c 'import sys, json; print(json.load(sys.stdin)["upload_url"])' <<< "$RESP")
MEDIA_ID=$(python3 -c 'import sys, json; print(json.load(sys.stdin)["media_id"])' <<< "$RESP")

curl -X PUT -H "Content-Type: image/jpeg" --data-binary "@$FILE" "$UPLOAD_URL"

echo "Uploaded for media_id: $MEDIA_ID"
