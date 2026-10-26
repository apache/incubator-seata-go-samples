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
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"seata.apache.org/seata-go/v2/pkg/client"
	seataSQL "seata.apache.org/seata-go/v2/pkg/datasource/sql"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

var db *sql.DB

func main() {
	client.InitPath("./conf/seatago.yml")
	var err error
	db, err = sql.Open(seataSQL.SeataXAMySQLDriver, mysqlDSN())
	if err != nil {
		panic(fmt.Sprintf("open database failed: %v", err))
	}
	defer db.Close()

	contextInstance := context.Background()

	testCommitUserId := "NO-XA-INSERT-COMMIT-101"
	testRollbackUserId := "NO-XA-INSERT-ROLLBACK-102"

	cleanupTestData(contextInstance, testCommitUserId)
	cleanupTestData(contextInstance, testRollbackUserId)

	runCommitTest(contextInstance, testCommitUserId)
	runRollbackTest(contextInstance, testRollbackUserId)

	cleanupTestData(contextInstance, testCommitUserId)
	cleanupTestData(contextInstance, testRollbackUserId)

	fmt.Println("xa insert commit and rollback integration test passed")
}

func runCommitTest(ctx context.Context, userId string) {
	transactionError := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "XATest_Insert_Commit",
		Timeout: time.Second * 30,
	}, func(txCtx context.Context) error {
		statement := "INSERT INTO order_tbl (user_id, commodity_code, count, money, descs) VALUES (?, ?, ?, ?, ?)"
		executionResult, executionError := db.ExecContext(txCtx, statement, userId, "C300001", 10, 100, "xa insert commit")
		if executionError != nil {
			return executionError
		}
		rows, rowsError := executionResult.RowsAffected()
		if rowsError != nil {
			return rowsError
		}
		if rows != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rows)
		}
		return nil
	})

	if transactionError != nil {
		panic(fmt.Sprintf("insert commit transaction failed: %v", transactionError))
	}

	var recordCount int64
	queryError := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM order_tbl WHERE user_id = ?", userId).Scan(&recordCount)
	if queryError != nil {
		panic(fmt.Sprintf("query committed record failed: %v", queryError))
	}
	if recordCount != 1 {
		panic(fmt.Sprintf("expected 1 record after commit, got %d", recordCount))
	}
}

func runRollbackTest(ctx context.Context, userId string) {
	mockBusinessError := errors.New("simulated business rollback trigger")

	transactionError := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "XATest_Insert_Rollback",
		Timeout: time.Second * 30,
	}, func(txCtx context.Context) error {
		statement := "INSERT INTO order_tbl (user_id, commodity_code, count, money, descs) VALUES (?, ?, ?, ?, ?)"
		_, executionError := db.ExecContext(txCtx, statement, userId, "C300002", 20, 200, "xa insert rollback")
		if executionError != nil {
			return executionError
		}
		return mockBusinessError
	})

	if transactionError == nil {
		panic("expected rollback error from global transaction, got nil")
	}

	var recordCount int64
	queryError := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM order_tbl WHERE user_id = ?", userId).Scan(&recordCount)
	if queryError != nil {
		panic(fmt.Sprintf("query rollback record failed: %v", queryError))
	}
	if recordCount != 0 {
		panic(fmt.Sprintf("expected 0 record after rollback, got %d", recordCount))
	}
}

func cleanupTestData(ctx context.Context, userId string) {
	_, _ = db.ExecContext(ctx, "DELETE FROM order_tbl WHERE user_id = ?", userId)
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
	config.MultiStatements = true
	return config.FormatDSN()
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
