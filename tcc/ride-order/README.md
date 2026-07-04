<!--
    Licensed to the Apache Software Foundation (ASF) under one or more
    contributor license agreements.  See the NOTICE file distributed with
    this work for additional information regarding copyright ownership.
    The ASF licenses this file to You under the Apache License, Version 2.0
    (the "License"); you may not use this file except in compliance with
    the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
    
    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
-->

# TCC Ride Order Sample

A ride-hailing order scenario demonstrating the TCC (Try-Confirm-Cancel) distributed
transaction pattern with five independent services coordinated by seata-go.

## Scenario

A passenger requests a ride. Creating a valid order requires five services to each
reserve their own resources first. Only when all reservations succeed does the order
get confirmed across all services. If any one fails, all already-reserved resources
must be explicitly released.

### Services

| Service | Port | Try (Reserve) | Confirm (Commit) | Cancel (Release) |
|---------|------|--------------|-------------------|------------------|
| **order-service** | 8001 | Create pending order | Mark order confirmed | Mark order cancelled |
| **dispatch-service** | 8002 | Reserve available driver | Mark driver busy | Release driver back to pool |
| **pricing-service** | 8003 | Lock estimated fare | Confirm fare lock | Release fare lock |
| **coupon-service** | 8004 | Freeze an available coupon | Mark coupon used | Unfreeze coupon |
| **capacity-service** | 8005 | Reserve vehicle slot (+1) | Keep reservation | Release slot (-1) |

The **initiator** (ride-order-service) acts as the Transaction Manager (TM) and
coordinates the full TCC lifecycle via HTTP calls to each service.

## Architecture

```
                         Seata TC (8091)
                        ╱  │  │  │  │  ╲
                       ╱   │  │  │  │   ╲
  initiator ──HTTP──► order dispatch pricing coupon capacity
  (TM)                :8001 :8002   :8003  :8004  :8005
                       │     │       │      │      │
                       └─────┴───────┴──────┴──────┘
                                  MySQL
```

- **Initiator** starts a global transaction and calls each service's `/prepare` endpoint via HTTP,
  passing the XID in the request header.
- Each **service** registers its TCC branch with Seata TC during Prepare.
- **Seata TC** calls Commit or Rollback on each service directly via the seata protocol (TCP).
- Each service has its own `seatago.yml` with a unique `application-id` so that TC can correctly
  route branch callbacks to the right service process.

## TCC Lifecycle

### Success Flow

```
initiator (TM)
    │
    ├── POST order-service/prepare       ✓ pending order created
    ├── POST pricing-service/prepare     ✓ fare locked
    ├── POST coupon-service/prepare      ✓ coupon frozen
    ├── POST capacity-service/prepare    ✓ slot reserved
    └── POST dispatch-service/prepare    ✓ driver reserved
    
    All Try succeeded → Seata TC triggers Confirm on all branches:
    
    ├── order-service.Confirm()      → order confirmed
    ├── pricing-service.Confirm()    → fare confirmed
    ├── coupon-service.Confirm()     → coupon used
    ├── capacity-service.Confirm()   → slot confirmed
    └── dispatch-service.Confirm()   → driver assigned
```

### Failure Flow (dispatch fails)

```
initiator (TM)
    │
    ├── POST order-service/prepare       ✓ pending order created
    ├── POST pricing-service/prepare     ✓ fare locked
    ├── POST coupon-service/prepare      ✓ coupon frozen
    ├── POST capacity-service/prepare    ✓ slot reserved
    └── POST dispatch-service/prepare    ✗ NO AVAILABLE DRIVER
    
    Try failed → Seata TC triggers Cancel on all registered branches:
    
    ├── order-service.Cancel()      → order cancelled
    ├── pricing-service.Cancel()    → fare released
    ├── coupon-service.Cancel()     → coupon unfrozen
    └── capacity-service.Cancel()   → slot released
    
    All resources restored to original state.
```

## Idempotency

All Confirm and Cancel methods are idempotent:

- **Status-based SQL conditions**: Each UPDATE uses `WHERE status=<expected>`, so repeated
  calls on an already-committed/cancelled record affect zero rows and return success.
- **In-memory deduplication**: Each service tracks active transactions in a `sync.Map` keyed
  by XID. Once a Confirm/Cancel completes, the entry is removed. Subsequent calls for the
  same XID find no entry and return success immediately.

> **Limitation (by design, to keep the sample small):** the XID→resource mapping lives in
> process memory. It makes Confirm/Cancel idempotent against TC retries and handles *empty
> rollback* (a branch whose Try failed), but it is **not crash-durable** and does **not** guard
> against TCC *suspension* (a delayed Try arriving after Cancel). If a service restarts between
> Try and the TC callback, the mapping is lost and the reserved resource is left stranded.
> Production code should use the seata-go **TCC fence** (`tcc_fence_log` table) instead — see the
> [`tcc/fence`](../fence) sample.

## How to Run

### 1. Start infrastructure

```bash
cd tcc/ride-order
docker-compose up -d
```

This starts MySQL (with auto-initialized `seata_ride` database) and Seata Server.

### 2. Start all five services (each in a separate terminal, from its own directory)

```bash
cd tcc/ride-order/order-service    && go run .   # :8001
cd tcc/ride-order/dispatch-service && go run .   # :8002
cd tcc/ride-order/pricing-service  && go run .   # :8003
cd tcc/ride-order/coupon-service   && go run .   # :8004
cd tcc/ride-order/capacity-service && go run .   # :8005
```

Each service must be started from its own directory so it can find its local `seatago.yml`.

### 3. Run the initiator

```bash
cd tcc/ride-order/initiator
go run .
```

### 4. Expected output

**Scenario 1** (success): The initiator logs five successful Prepare calls. Each
service logs `[*-Try]` followed by `[*-Confirm]`.

**Scenario 2** (dispatch failure): The initiator logs four successful Prepare calls,
then `dispatch-service prepare failed`. Each of the four services logs `[*-Cancel]`,
restoring all resources.

### Reset between runs

Each successful ride consumes real resources (a coupon becomes *used*, a driver becomes
*busy*) — that is the correct business outcome, so those rows are **not** restored on Confirm.
The seed data (6 coupons, 6 drivers) is enough for several runs. To start completely fresh,
recreate the database volume:

```bash
docker-compose down -v && docker-compose up -d
```

### Customize MySQL connection

Override via environment variables (set on each service):

```bash
export MYSQL_HOST=127.0.0.1
export MYSQL_PORT=3306
export MYSQL_USERNAME=root
export MYSQL_PASSWORD=12345678
export MYSQL_DB=seata_ride
```
