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

# Insurance Claim Saga Sample

This sample demonstrates how to migrate a legacy long-running insurance claim process to Seata Go Saga.

The workflow contains the following steps:

1. Verify claimant identity
2. Create damage assessment record
3. Reserve payout funds
4. Notify the assigned surveyor
5. Execute bank transfer

The legacy implementation is a sequential flow. If the first four steps succeed but the bank transfer fails in step five, the system is left with intermediate state such as verified identity, created assessment, reserved funds, and sent surveyor notification, with no automatic rollback.

After migrating to Saga, each step is mapped to a forward action and a compensating action:

| Forward Action | Compensating Action |
| --- | --- |
| VerifyIdentity | UnverifyClaim |
| CreateDamageAssessment | DeleteDamageAssessment |
| ReservePayoutFunds | ReleasePayoutFunds |
| NotifyAssignedSurveyor | CancelSurveyorNotification |
| ExecuteBankTransfer | None |

When `ExecuteBankTransfer` fails, Saga compensates in reverse order:

1. CancelSurveyorNotification
2. ReleasePayoutFunds
3. DeleteDamageAssessment
4. UnverifyClaim

## Directory Layout

- `legacy/`: the legacy sequential implementation used to show the pre-migration problem
- `orchestrator/`: the Saga orchestrator starter
- `services/`: five independent Go HTTP services
- `statelang/insurance_claim_saga.json`: Saga state machine definition
- `sql/mysql_claim_saga_schema.sql`: Saga persistence tables and business demo tables
- `docker-compose.yml`: MySQL and Seata Server

## Start the Infrastructure

```bash
cd saga/insurance_claim
docker-compose up -d
```

Default ports:

- MySQL: `3306`
- Seata Server: `8091`
- identity service: `18081`
- assessment service: `18082`
- funds service: `18083`
- surveyor service: `18084`
- transfer service: `18085`

## Start the Five Services

Open five terminals from the repository root and run:

```bash
go run ./saga/insurance_claim/services/identity
go run ./saga/insurance_claim/services/assessment
go run ./saga/insurance_claim/services/funds
go run ./saga/insurance_claim/services/surveyor
go run ./saga/insurance_claim/services/transfer
```

## Run the Legacy Sequential Flow

Success case:

```bash
go run ./saga/insurance_claim/legacy
```

Simulate a bank transfer failure:

```bash
go run ./saga/insurance_claim/legacy -failTransfer
```

When the flow fails, it stops immediately and prints the current business snapshot. You will see that the intermediate state from the first four steps is still present, which is exactly the problem with the legacy implementation.

## Run the Migrated Saga Flow

Success case:

```bash
go run ./saga/insurance_claim/orchestrator
```

Simulate a bank transfer failure:

```bash
go run ./saga/insurance_claim/orchestrator -failTransfer
```

The failure case prints:

- `xid`
- final Saga status
- compensation status
- business snapshot
- `actionTrail`

Focus on `actionTrail`. It shows:

1. The first four forward actions execute
2. The bank transfer fails
3. The four compensating actions execute in reverse order

## Implementation Notes

- The five services run as independent processes and expose forward and compensating actions over HTTP
- The Saga orchestrator uses the built-in seata-go `http` invoker to call those services
- The state machine uses `CompensateState` to map each forward action to its compensating action
- The `failTransfer` parameter provides a stable way to reproduce the bank transfer failure
- `claim_step_log` records both forward and compensation execution order for easy observation

## Debug with a Local seata-go Checkout

If you want to debug this sample against a local `../incubator-seata-go` checkout:

```bash
go mod edit -replace seata.apache.org/seata-go=../incubator-seata-go
go run ./saga/insurance_claim/orchestrator -failTransfer
go mod edit -dropreplace seata.apache.org/seata-go
```
