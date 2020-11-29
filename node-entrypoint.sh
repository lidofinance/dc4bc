#!/bin/bash

SHARED_DIR="/go/src/shared"
KEY_STORE_DIR="$SHARED_DIR/dc4bc_${USERNAME}_key_store"
STATE_DIR="$SHARED_DIR/dc4bc_${USERNAME}_state"

if [ ! -d "$KEY_STORE_DIR" ]; then
  echo "Keystore is not found, generating new keys"
  ./dc4bc_d gen_keys --username "$USERNAME" --key_store_dbdsn "$KEY_STORE_DIR"
fi

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