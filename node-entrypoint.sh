#!/bin/bash

SHARED_DIR="/go/src/shared"
KEY_STORE_DIR="$SHARED_DIR/dc4bc_${USERNAME}_key_store"
STATE_DIR="$SHARED_DIR/dc4bc_${USERNAME}_state"
QR_SCANNER_PORT=9090

if [ ! -d "$KEY_STORE_DIR" ]; then
  ./dc4bc_d gen_keys --username "$USERNAME" --key_store_dbdsn "$KEY_STORE_DIR"
fi

python3 -m http.server $QR_SCANNER_PORT > /dev/null 2>&1 &
echo "Started QR scanner. Go to http://localhost:$QR_SCANNER_PORT/qr/index.html"

./dc4bc_d start \
  --username "$USERNAME" \
  --key_store_dbdsn "$KEY_STORE_DIR" \
  --listen_addr localhost:8080 \
  --state_dbdsn "$STATE_DIR" \
  --storage_dbdsn "$STORAGE_DBDSN" \
  --producer_credentials producer:producerpass \
  --consumer_credentials consumer:consumerpass \
  --kafka_truststore_path ./ca.crt \
  --storage_topic "$STORAGE_TOPIC"