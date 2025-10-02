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

# TCC RocketMQ Sample

This sample demonstrates how to use Seata-Go with RocketMQ in TCC mode.

## Architecture

This sample shows:
- Using TCC pattern with RocketMQ transactional messages
- Integrating Seata distributed transaction with RocketMQ
- Automatic transaction coordination between TCC services and MQ messages

## Components

- **OrderService**: A TCC service that manages order operations
- **RocketMQ Producer**: Sends transactional messages coordinated with Seata global transaction

## Prerequisites

- Docker and Docker Compose installed
- Go 1.20+ installed
- Port 8091 (Seata), 9876 (RocketMQ NameServer), 10911 (RocketMQ Broker) available

## Quick Start

### One-Command Run

Simply execute the run script:

```bash
./run.sh
```

This will:
1. Start Seata Server via Docker Compose
2. Start RocketMQ NameServer and Broker via Docker Compose
3. Wait for all services to be ready
4. Run the TCC RocketMQ sample

### Stop Services

To stop all Docker services:

```bash
./stop.sh
```

## Manual Run

If you prefer to run manually:

1. Start services:
   ```bash
   cd ../../../dockercompose
   docker-compose up -d seata-server rocketmq-namesrv rocketmq-broker
   ```

2. Wait for services to be ready (check with `docker-compose ps`)

3. Run the sample:
   ```bash
   cd cmd
   go run main.go
   ```

## What Happens

1. Global transaction starts
2. OrderService.Prepare() executes (TCC prepare phase)
3. RocketMQ message is sent as half message (prepare phase)
4. If both succeed, global transaction commits:
   - OrderService.Commit() is called
   - RocketMQ message is committed and becomes visible to consumers
5. If any fails, global transaction rolls back:
   - OrderService.Rollback() is called
   - RocketMQ message is rolled back and discarded

## Configuration

- Seata config: `../../../conf/seatago.yml`
- RocketMQ config: `conf/rocketmq.yml`
- Docker Compose: `../../../dockercompose/docker-compose.yml`

## Troubleshooting

- If services fail to start, check Docker logs: `docker-compose logs <service-name>`
- Ensure ports 8091, 9876, 10911 are not in use
- Check if Docker daemon is running

