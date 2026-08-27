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

package common

import (
	"database/sql"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func InitDB() {
	defaultEnv()
	dsn := os.ExpandEnv("${MYSQL_USERNAME}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}?parseTime=true")
	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		panic("open mysql failed: " + err.Error())
	}
	if err = DB.Ping(); err != nil {
		panic("ping mysql failed: " + err.Error())
	}
}

func defaultEnv() {
	envDefaults := map[string]string{
		"MYSQL_HOST":     "127.0.0.1",
		"MYSQL_PORT":     "3306",
		"MYSQL_USERNAME": "root",
		"MYSQL_PASSWORD": "12345678",
		"MYSQL_DB":       "seata_ride",
	}
	for k, v := range envDefaults {
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}
