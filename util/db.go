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

package util

import (
	"database/sql"
	"os"

	sql2 "seata.apache.org/seata-go/pkg/datasource/sql"
)

func GetAtMySqlDb() *sql.DB {
	defaultEnv()
	dsn := os.ExpandEnv("${MYSQL_USERNAME}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}?multiStatements=true&interpolateParams=true")
	dbAt, err := sql.Open(sql2.SeataATMySQLDriver, dsn)
	if err != nil {
		panic("init seata at mysql driver error")
	}
	return dbAt
}

func GetXAMySqlDb() *sql.DB {
	defaultEnv()
	dsn := os.ExpandEnv("${MYSQL_USERNAME}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}?multiStatements=true&interpolateParams=true")
	dbAt, err := sql.Open(sql2.SeataXAMySQLDriver, dsn)
	if err != nil {
		panic("init seata at mysql driver error")
	}
	return dbAt
}

func defaultEnv() {
	if os.Getenv("MYSQL_HOST") == "" {
		os.Setenv("MYSQL_HOST", "127.0.0.1")
	}
	if os.Getenv("MYSQL_PORT") == "" {
		os.Setenv("MYSQL_PORT", "3306")
	}
	if os.Getenv("MYSQL_USERNAME") == "" {
		os.Setenv("MYSQL_USERNAME", "root")
	}
	if os.Getenv("MYSQL_PASSWORD") == "" {
		os.Setenv("MYSQL_PASSWORD", "123456")
	}
	if os.Getenv("MYSQL_DB") == "" {
		os.Setenv("MYSQL_DB", "seata_client")
	}
}
