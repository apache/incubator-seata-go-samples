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
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"seata.apache.org/seata-go/v2/pkg/client"
	seataSQL "seata.apache.org/seata-go/v2/pkg/datasource/sql"
)

const (
	caseTimeout = 30 * time.Second
	pollTimeout = 15 * time.Second
)

type batchSuite struct {
	db *sql.DB
}

func main() {
	client.InitPath("./conf/seatago.yml")

	db, err := sql.Open(seataSQL.SeataATMySQLDriver, mysqlDSN())
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	suite := batchSuite{db: db}
	if err := suite.runManagedSuccess(ctx); err != nil {
		log.Fatalf("managed success: %v", err)
	}
	if err := suite.runManagedPartialFailure(ctx); err != nil {
		log.Fatalf("managed partial failure: %v", err)
	}
	if err := suite.runCallerOwnedGlobalRollback(ctx); err != nil {
		log.Fatalf("caller-owned global rollback: %v", err)
	}
	log.Println("AT semantic batch integration case passed")
}

func mysqlDSN() string {
	if dsn := os.Getenv("MYSQL_DSN"); dsn != "" {
		return dsn
	}

	password := os.Getenv("MYSQL_PASSWORD")
	if password == "" {
		password = envOrDefault("MYSQL_ROOT_PASSWORD", "12345678")
	}
	config := mysqlDriver.NewConfig()
	config.User = envOrDefault("MYSQL_USERNAME", "root")
	config.Passwd = password
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(envOrDefault("MYSQL_HOST", "127.0.0.1"), envOrDefault("MYSQL_PORT", "3306"))
	config.DBName = envOrDefault("MYSQL_DB", "seata_client")
	config.InterpolateParams = true
	return config.FormatDSN()
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
