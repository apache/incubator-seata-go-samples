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

if [ -z "$1" ]; then
  echo "Provide test directory please, like : ./integrate_test.sh helloworld"
  exit
fi

P_DIR=$(pwd)/$1
#if [ -f "$P_DIR"/build/test.sh ]; then
#    "$P_DIR"/build/test.sh "$P_DIR"
#    result=$?
#    exit $((result))
#fi

INTEGRATE_DIR=$(pwd)/$1

echo "$INTEGRATE_DIR"



# waiting for port release
sleep 5

# start server
make -f "$INTEGRATE_DIR"/Makefile run DIRECTORY=$INTEGRATE_DIR
result=$?


exit $((result))