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

SEATA_CONF=${1:-saga/e2e/seatago.yaml}
ENGINE_CONF=${2:-saga/e2e/config.yaml}

echo "Running saga e2e with seataConf=$SEATA_CONF engineConf=$ENGINE_CONF"
go run ./saga/e2e -seataConf="$SEATA_CONF" -engineConf="$ENGINE_CONF"
