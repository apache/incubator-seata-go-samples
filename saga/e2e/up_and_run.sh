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

DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

echo "Resetting docker-compose services (MySQL + Seata Server)..."
docker-compose -f "$DIR/docker-compose.yml" down -v --remove-orphans || true
docker-compose -f "$DIR/docker-compose.yml" rm -f -s -v || true
echo "Starting docker-compose services (fresh)..."
docker-compose -f "$DIR/docker-compose.yml" up -d --force-recreate

echo "Waiting for MySQL (3306) and Seata Server (8091) to be ready..."
# simple wait loop
for i in {1..60}; do
  nc -z 127.0.0.1 3306 && nc -z 127.0.0.1 8091 && break || true
  sleep 2
done

echo "Running saga e2e example"
go run ./saga/e2e -seataConf="$DIR/seatago.yaml" -engineConf="$DIR/config.yaml"
