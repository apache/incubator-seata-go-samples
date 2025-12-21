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

.PHONY: run migrate

SEATA_CONF ?= saga/e2e/seatago.yaml
ENGINE_CONF ?= saga/e2e/config.yaml

run:
	go run ./saga/e2e -seataConf=$(SEATA_CONF) -engineConf=$(ENGINE_CONF)

# Requires mysql client; provide MYSQL_HOST, MYSQL_PORT, MYSQL_USER, MYSQL_PWD, MYSQL_DB
migrate:
	@[ -n "$(MYSQL_HOST)" ] || (echo "MYSQL_HOST is required" && exit 1)
	@[ -n "$(MYSQL_PORT)" ] || (echo "MYSQL_PORT is required" && exit 1)
	@[ -n "$(MYSQL_USER)" ] || (echo "MYSQL_USER is required" && exit 1)
	@[ -n "$(MYSQL_PWD)" ] || (echo "MYSQL_PWD is required" && exit 1)
	@[ -n "$(MYSQL_DB)" ] || (echo "MYSQL_DB is required" && exit 1)
	@echo "Applying schema to $(MYSQL_HOST):$(MYSQL_PORT)/$(MYSQL_DB)..."
	@mysql -h"$(MYSQL_HOST)" -P"$(MYSQL_PORT)" -u"$(MYSQL_USER)" -p"$(MYSQL_PWD)" "$(MYSQL_DB)" < sql/mysql_saga_schema.sql
	@echo "Schema applied."
