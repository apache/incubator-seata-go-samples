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

# RocketMQ TCC Sample

This sample demonstrates how to use Seata-Go's RocketMQ integration for distributed transactional messaging.

## Use Case Description

This sample showcases:
- Sending RocketMQ transactional messages within Seata global transactions
- Automatic TCC (Try-Confirm-Cancel) handling by the SDK
- Both commit and rollback scenarios

## Key Teaching Point

**The RocketMQ integration has TCC built-in.** When you call `producer.Send(ctx, msg)` inside a global transaction, the SDK automatically:
1. Detects the global transaction context via `tm.IsGlobalTx(ctx)`
2. Invokes its internal TCC proxy to register a branch transaction
3. Sends a half-message to RocketMQ (Prepare phase)
4. Delegates Commit/Rollback to RocketMQ's transaction listener based on global transaction outcome

**You do NOT need to create a TCC service wrapper** - just use the producer directly.

## Architecture

```
main.go
  ↓
tm.WithGlobalTx() ← Global Transaction Boundary
  ↓
producer.Send(ctx, msg) ← SDK auto-applies TCC
  ↓
[SDK Internal] tccProxy.Prepare() → RocketMQ half-message
  ↓
[On Commit] → TransactionListener → Message becomes consumable
[On Rollback] → TransactionListener → Message deleted
```

## Prerequisites

1. **Seata TC Server**
   - Version: Compatible with seata-go v2.x
   - Address: `127.0.0.1:8091`

2. **RocketMQ**
   - Version: 4.x or 5.x
   - NameServer: `127.0.0.1:9876`
   - Broker running

## Setup Steps

### 1. Start Infrastructure

```bash
# Start Seata Server and RocketMQ using docker-compose
cd ../../dockercompose
docker-compose up -d
```

### 2. Run the Sample

**Commit Scenario** (message will be sent and committed):
```bash
cd cmd
go run main.go --mode=commit
```

**Rollback Scenario** (message will be sent but rolled back):
```bash
cd cmd
go run main.go --mode=rollback
```

## Expected Behavior

### Commit Mode
1. Application starts global transaction (XID logged)
2. `producer.Send()` is called → SDK internally calls `tccProxy.Prepare()`
3. RocketMQ half-message sent (not consumable yet)
4. Business function returns `nil` → global transaction commits
5. RocketMQ TransactionListener receives commit signal
6. Message becomes consumable
7. Consumers can now receive the message

### Rollback Mode
1. Application starts global transaction (XID logged)
2. `producer.Send()` is called → SDK internally calls `tccProxy.Prepare()`
3. RocketMQ half-message sent
4. Business function returns `error` → global transaction rolls back
5. RocketMQ TransactionListener receives rollback signal
6. Message is deleted/canceled
7. Message never becomes consumable

## Verification

Check Seata TC server logs for branch registration:
```
Branch registered: xid=..., branchId=..., resourceId=RocketMQTCC
```

Check RocketMQ console or CLI tools to verify message visibility:
- Commit mode: Message appears in topic `seata-tcc-test`
- Rollback mode: No message in topic

## Code Walkthrough

```go
// Create producer - SDK creates internal TCC proxy
producer, _ := rocketmq.NewSeataMQProducer(cfg)
producer.Start()

// Execute global transaction
tm.WithGlobalTx(ctx, config, func(ctx context.Context) error {
    // Direct SDK usage - no wrapper needed
    msg := primitive.NewMessage("topic", payload)
    _, err := producer.Send(ctx, msg)  // SDK handles TCC automatically
    
    if mode == "rollback" {
        return fmt.Errorf("trigger rollback")  // Error triggers rollback
    }
    return nil  // Success triggers commit
})
```

## Troubleshooting

**Issue**: `seata server not available`
- Check Seata TC server is running on `127.0.0.1:8091`
- Verify `conf/seatago.yml` configuration

**Issue**: `RocketMQ connection failed`
- Ensure RocketMQ NameServer and Broker are running
- Check network connectivity to `127.0.0.1:9876`

**Issue**: `params must be *primitive.Message`
- This error indicates incorrect SDK usage
- Always pass `*primitive.Message` to `producer.Send()`, not other types
