#!/usr/bin/env bash
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

# One-click e2e: optional docker-compose up + readiness wait, then run all scenarios.

DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
REPO_ROOT="$DIR/../.."

SEATA_CONF=${SEATA_CONF:-"$DIR/seatago.yaml"}
ENGINE_CONF=${ENGINE_CONF:-"$DIR/config.yaml"}
DO_UP=${DO_UP:-"false"}
# Wait settings (seconds)
WAIT_TIMEOUT=${WAIT_TIMEOUT:-60}
WAIT_INTERVAL=${WAIT_INTERVAL:-2}
# Extra margin after MySQL becomes reachable (to avoid initial EOF on fresh start)
WAIT_READY_MARGIN=${WAIT_READY_MARGIN:-15}

usage() {
  cat <<EOF
Usage: $(basename "$0") [--up] [--seata <seatago.yaml>] [--engine <config.yaml>]

Options:
  --up                 Start docker-compose (MySQL + Seata Server) before running
  --seata <file>       Path to seatago.yaml (default: $SEATA_CONF)
  --engine <file>      Path to engine config (default: $ENGINE_CONF)

Runs three scenarios in order:
  1) success
  2) compensate-balance
  3) compensate-inventory
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --up) DO_UP="true"; shift ;;
    --seata) SEATA_CONF="$2"; shift 2 ;;
    --engine) ENGINE_CONF="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown arg: $1"; usage; exit 1 ;;
  esac
done

require() { command -v "$1" >/dev/null 2>&1 || { echo "Missing required command: $1"; exit 1; }; }

echo "[+] Repo root: $REPO_ROOT"
cd "$REPO_ROOT"

require go

[[ -f "$SEATA_CONF" ]] || { echo "seatago.yaml not found: $SEATA_CONF"; exit 1; }
[[ -f "$ENGINE_CONF" ]] || { echo "engine config not found: $ENGINE_CONF"; exit 1; }

if [[ "$DO_UP" == "true" ]]; then
  require docker-compose || require docker
  echo "[+] Resetting docker-compose services (MySQL + Seata Server) ..."
  # Stop and remove old containers, networks and volumes to ensure a clean start
  docker-compose -f "$DIR/docker-compose.yml" down -v --remove-orphans || true
  docker-compose -f "$DIR/docker-compose.yml" rm -f -s -v || true
  echo "[+] Starting docker-compose services (fresh) ..."
  docker-compose -f "$DIR/docker-compose.yml" up -d --force-recreate
fi

wait_for_tcp() {
  local host="$1"; local port="$2"; local name="$3"; local timeout="${4:-$WAIT_TIMEOUT}"; local interval="${5:-$WAIT_INTERVAL}"
  echo "[+] Waiting for $name at $host:$port (timeout=${timeout}s) ..."
  local start_ts=$(date +%s)
  while true; do
    if command -v nc >/dev/null 2>&1; then
      if nc -z "$host" "$port" 2>/dev/null; then echo "[+] $name is reachable"; return 0; fi
    else
      (exec 3<>/dev/tcp/$host/$port) 2>/dev/null && { exec 3>&- || true; echo "[+] $name is reachable"; return 0; }
    fi
    local now=$(date +%s)
    if (( now - start_ts >= timeout )); then
      echo "[-] Timeout waiting for $name at $host:$port"; return 1
    fi
    sleep "$interval"
  done
}

# Parse Seata target from seatago.yaml
SEATA_ADDR=$(awk '
  /grouplist:/ { gl=1; next }
  gl==1 && /^[^[:space:]]/ { gl=0 }
  gl==1 && /default:/ {
    line=$0; gsub(/"/,"",line); sub(/.*default:[[:space:]]*/,"",line); print line; exit;
  }
' "$SEATA_CONF" 2>/dev/null || true)
[[ -n "$SEATA_ADDR" ]] || { echo "[-] Could not extract Seata service.grouplist.default from $SEATA_CONF"; exit 1; }
SEATA_HOST=${SEATA_ADDR%%:*}
SEATA_PORT=${SEATA_ADDR##*:}

# Parse MySQL from engine store_dsn (root:pwd@tcp(127.0.0.1:3306)/db)
MYSQL_ADDR=$(sed -nE 's/.*store_dsn:[[:space:]]*"?[^\(]*tcp\(([^)]*)\).*/\1/p' "$ENGINE_CONF" | head -n1)
MYSQL_HOST=${MYSQL_ADDR%%:*}
MYSQL_PORT=${MYSQL_ADDR##*:}

if [[ "$DO_UP" == "true" ]]; then
  if [[ -n "${MYSQL_HOST:-}" && -n "${MYSQL_PORT:-}" ]]; then
    wait_for_tcp "$MYSQL_HOST" "$MYSQL_PORT" "MySQL" || exit 1
  fi
  wait_for_tcp "$SEATA_HOST" "$SEATA_PORT" "Seata" || exit 1
  echo "[+] Extra wait ${WAIT_READY_MARGIN}s for services to finish initialization ..."
  sleep "$WAIT_READY_MARGIN"
else
  wait_for_tcp "$SEATA_HOST" "$SEATA_PORT" "Seata" || exit 1
fi

echo "[+] Running all scenarios via single process ..."
OUT=$(go run ./saga/e2e -seataConf="$SEATA_CONF" -engineConf="$ENGINE_CONF" | tee /dev/stderr)

XID1=$(echo "$OUT" | awk -F '[ =,]+' '/SCENARIO success XID=/{print $4}' | tail -n1)
[[ -n "$XID1" ]] || { echo "[-] could not extract XID for success"; exit 1; }
echo "[+] Validating DB for success (XID=$XID1) ..."
go run ./saga/e2e/dbcheck -engine "$ENGINE_CONF" -xid "$XID1" -scenario success

XID2=$(echo "$OUT" | awk -F '[ =,]+' '/SCENARIO compensate-balance XID=/{print $4}' | tail -n1)
[[ -n "$XID2" ]] || { echo "[-] could not extract XID for compensate-balance"; exit 1; }
echo "[+] Validating DB for compensate-balance (XID=$XID2) ..."
go run ./saga/e2e/dbcheck -engine "$ENGINE_CONF" -xid "$XID2" -scenario compensate-balance

XID3=$(echo "$OUT" | awk -F '[ =,]+' '/SCENARIO compensate-inventory XID=/{print $4}' | tail -n1)
[[ -n "$XID3" ]] || { echo "[-] could not extract XID for compensate-inventory"; exit 1; }
echo "[+] Validating DB for compensate-inventory (XID=$XID3) ..."
go run ./saga/e2e/dbcheck -engine "$ENGINE_CONF" -xid "$XID3" -scenario compensate-inventory

echo "[+] All e2e scenarios finished"
exit 0
