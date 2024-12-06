#
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
#

#!/bin/bash

echo "Checking Seata server..."
res=1
passCode=0
while [ "$res" != "$passCode" ]; do
  echo "Waiting for Seata server..."
  sleep 5
  curl -s -o /dev/null 127.0.0.1:7091
  res=$?
done

echo "Seata server is up!"

CONTAINER_NAME="mysql"
MYSQL_USER="root"
MYSQL_PASS="12345678"

echo "Checking MySQL container..."
res=1
while [ "$res" != "0" ]; do
  if docker exec "$CONTAINER_NAME" mysql -u "$MYSQL_USER" -p"$MYSQL_PASS" -e "SELECT 1;" >/dev/null 2>&1; then
      res=0
  else
      echo "Waiting for MySQL container..."
      sleep 5
  fi
done

echo "MySQL container is up!"

