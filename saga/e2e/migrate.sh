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

if [ -z "${MYSQL_HOST:-}" ] || [ -z "${MYSQL_PORT:-}" ] || [ -z "${MYSQL_USER:-}" ] || [ -z "${MYSQL_PWD:-}" ] || [ -z "${MYSQL_DB:-}" ]; then
  echo "Please set MYSQL_HOST, MYSQL_PORT, MYSQL_USER, MYSQL_PWD, MYSQL_DB"
  exit 1
fi

echo "Applying schema to $MYSQL_HOST:$MYSQL_PORT/$MYSQL_DB ..."
mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PWD" "$MYSQL_DB" < saga/e2e/sql/mysql_saga_schema.sql
echo "Schema applied."
