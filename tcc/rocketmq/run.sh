#!/bin/bash
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


set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCKER_COMPOSE_DIR="$PROJECT_ROOT/dockercompose"

echo "=========================================="
echo "  Seata-Go RocketMQ TCC Sample Runner"
echo "=========================================="
echo ""

echo "Step 1: Starting Docker services..."
cd "$DOCKER_COMPOSE_DIR"
docker-compose up -d seata-server rocketmq-namesrv rocketmq-broker
echo "✓ Docker services started"
echo ""

echo "Step 2: Waiting for services to be ready..."
echo -n "  - Waiting for Seata Server (8091)..."
timeout=60
count=0
while ! nc -z localhost 8091 > /dev/null 2>&1; do
    sleep 1
    count=$((count + 1))
    if [ $count -ge $timeout ]; then
        echo " TIMEOUT!"
        echo "Error: Seata Server failed to start within ${timeout}s"
        exit 1
    fi
done
echo " Ready!"

echo -n "  - Waiting for RocketMQ NameServer (9876)..."
count=0
while ! nc -z localhost 9876 > /dev/null 2>&1; do
    sleep 1
    count=$((count + 1))
    if [ $count -ge $timeout ]; then
        echo " TIMEOUT!"
        echo "Error: RocketMQ NameServer failed to start within ${timeout}s"
        exit 1
    fi
done
echo " Ready!"

echo -n "  - Waiting for RocketMQ Broker (10911)..."
sleep 3
count=0
while ! nc -z localhost 10911 > /dev/null 2>&1; do
    sleep 1
    count=$((count + 1))
    if [ $count -ge $timeout ]; then
        echo " TIMEOUT!"
        echo "Error: RocketMQ Broker failed to start within ${timeout}s"
        exit 1
    fi
done
echo " Ready!"
echo ""

echo "Step 3: Running RocketMQ TCC sample..."
cd "$SCRIPT_DIR/cmd"
go run main.go
