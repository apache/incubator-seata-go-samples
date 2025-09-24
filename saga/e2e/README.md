<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

# Saga E2E (MySQL) — End‑to‑End Guide

Chinese version: see `README_zh.md`.

This example demonstrates a Java‑aligned Seata SAGA flow in Go, with MySQL persistence and Seata Server (TC) integration. It includes one‑click scripts, DB validation tooling, and compensation semantics compatible with the Java reference.

## What it starts

- MySQL 8.0 — stores SAGA state
- Seata Server 1.6.1 — TC for global coordination
- A demo SAGA: ReduceInventory → ReduceBalance (with compensations)

## Quick start (one‑click)

Prerequisites: Go 1.20+, Docker, and docker-compose available on the host.

```
saga/e2e/run_all.sh --up \
  --seata saga/e2e/seatago.yaml \
  --engine saga/e2e/config.yaml
```

## Use local seata-go while developing

The sample repository currently depends on a released version of seata-go. When
you need to exercise the E2E flow against your working tree at
`../incubator-seata-go`, temporarily add a replace directive before running:

```
go mod edit -replace github.com/seata/seata-go=../incubator-seata-go
go mod tidy # optional, only if dependencies changed
```

After the test run, drop the replace so the module file stays clean:

```
go mod edit -dropreplace github.com/seata/seata-go
```

When using Go 1.18+, you can alternatively create a temporary workspace:

```
go work init
go work use .
go work use ../incubator-seata-go
go run ./saga/e2e
rm go.work
```

Knobs:
- `WAIT_TIMEOUT` (default 60), `WAIT_INTERVAL` (2) TCP readiness
- `WAIT_READY_MARGIN` (default 15) extra wait after ports are open

Expected:
- `DB validation (success) OK`
- `DB validation (compensate-balance) OK`
- `DB validation (compensate-inventory) OK`
- `[+] All e2e scenarios finished`

## Configuration

- Seata client (`saga/e2e/seatago.yaml`)
  - `application-id`, `tx-service-group`
  - `service.grouplist.default`: `host:port` of Seata Server

- Saga engine (`saga/e2e/config.yaml`)
  - `store_enabled: true`, `store_type: mysql`
  - `store_dsn: user:pass@tcp(127.0.0.1:3306)/seata_saga?parseTime=true`
  - `tc_enabled: true`
  - `state_machine_resources: [saga/e2e/statelang/*.json]`

## Run individual scenarios

- Start + run (fresh): `saga/e2e/up_and_run.sh`
- Success only: `saga/e2e/run.sh [seatago.yaml] [config.yaml]`
- Compensation: `saga/e2e/run_compensation.sh [seatago.yaml] [config.yaml] [compensate-balance|compensate-inventory]`

## DB validation

```
go run ./saga/e2e/dbcheck \
  -engine saga/e2e/config.yaml \
  -xid <XID> \
  -scenario success|compensate-balance|compensate-inventory
```

Checks machine row end state and per‑state rows with compensation linkage.

## Schema migration

```
MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 \
MYSQL_USER=root MYSQL_PWD=secret MYSQL_DB=seata_saga \
saga/e2e/migrate.sh
```

## Troubleshooting

- Seata unreachable → check `service.grouplist.default`
- Branch register fails → ensure `application-id#tx-service-group` mapping exists in TC
- MySQL initial EOF/bad connection → increase `WAIT_READY_MARGIN`
- Optimistic‑lock 0‑row finish on success → benign; handled idempotently (debug only)
- TC GlobalReport timeouts → logged, do not fail local finish; DB validation still passes

## Pointers

- StateLang JSON: `statelang/reduce_inventory_and_balance.json`
- Engine: `pkg/saga/statemachine/engine/pcext/*`, store: `pkg/saga/statemachine/store/db/statelog.go`
- Scripts: `run_all.sh`, `up_and_run.sh`, `run.sh`, `run_compensation.sh`
