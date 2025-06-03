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

# tests
array+=("integrate_test/at/insert")




DOCKER_DIR=$(pwd)/dockercompose
docker-compose -f $DOCKER_DIR/docker-compose.yml up -d

bash -f $DOCKER_DIR/docker-health-check.sh

for var in ${array[@]}; do
	./integrate_test.sh "$var"
	result=$?
	if [ $result -gt 0 ]; then
	      docker-compose -f $DOCKER_DIR/docker-compose.yml down
	      echo "test failed"
	      echo "$var"
        exit $result
	fi
done
docker-compose -f $DOCKER_DIR/docker-compose.yml down
