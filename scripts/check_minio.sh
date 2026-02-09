#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Usage: $0 <media_id>"
  exit 1
fi

MEDIA_ID="$1"
NETWORK="${SYS_DESIGN_NETWORK:-sys-design_default}"

# Use MinIO client in a throwaway container to list objects.
# This avoids exposing credentials and works with the docker network.

docker run --rm --network "$NETWORK" \
  -e MC_HOST_local="http://minioadmin:minioadmin@minio:9000" \
  minio/mc:latest ls "local/media/media/${MEDIA_ID}/" | sed -n '1,20p'
