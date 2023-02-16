/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"database/sql"

	sql2 "github.com/seata/seata-go/pkg/datasource/sql"
)

var (
	db *sql.DB
)

func initService() {
	var err error
	db, err = sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:6666)/seata_client?multiStatements=true&interpolateparams=true")
	if err != nil {
		panic("init service error")
	}
}
