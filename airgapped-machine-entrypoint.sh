#!/bin/bash

SHARED_DIR="/go/src/shared"
AIRGAPPED_STATE_DIR="$SHARED_DIR/dc4bc_${USERNAME}_airgapped_state"

./dc4bc_airgapped --db_path "$AIRGAPPED_STATE_DIR" \
  --password_expiration "$PASSWORD_EXPIRATION"