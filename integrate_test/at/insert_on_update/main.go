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
	"fmt"
	"time"

	"seata.apache.org/seata-go/pkg/client"
	sql2 "seata.apache.org/seata-go/pkg/datasource/sql"
	"seata.apache.org/seata-go/pkg/tm"
)

var db *sql.DB

func main() {
	client.InitPath("./conf/seatago.yml")
	initDB()

	ctx := context.Background()

	// execute insert on update within global transaction
	err := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx_InsertOnUpdate",
		Timeout: time.Second * 30,
	}, insertOnUpdateData)
	if err != nil {
		panic(fmt.Sprintf("transaction failed: %v", err))
	}

	// verify data was updated correctly
	if err := verifyData(ctx); err != nil {
		panic(fmt.Sprintf("data verification failed: %v", err))
	}

	// wait for undo log cleanup
	time.Sleep(time.Second * 10)

	// verify undo log is cleaned
	if err := verifyUndoLogCleaned(ctx); err != nil {
		panic(fmt.Sprintf("undo log verification failed: %v", err))
	}

	fmt.Println("insert_on_update test passed")
}

// insertOnUpdateData tests INSERT ... ON DUPLICATE KEY UPDATE
// The initial data has id=1, so this will trigger an UPDATE
func insertOnUpdateData(ctx context.Context) error {
	sql := "INSERT INTO order_tbl (id, user_id, commodity_code, count, money, descs) " +
		"VALUES (?, ?, ?, ?, ?, ?) " +
		"ON DUPLICATE KEY UPDATE count = ?, descs = ?"

	result, err := db.ExecContext(ctx, sql,
		1, "NO-100001", "C100000", 200, 10, "insert desc",
		200, "updated by insert_on_update")
	if err != nil {
		return fmt.Errorf("insert on update failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected failed: %w", err)
	}
	fmt.Printf("insert on update affected rows: %d\n", rows)

	return nil
}

func verifyData(ctx context.Context) error {
	var count int64
	var descs string

	err := db.QueryRowContext(ctx,
		"SELECT count, descs FROM order_tbl WHERE id = ?", 1).Scan(&count, &descs)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	if count != 200 {
		return fmt.Errorf("expected count=200, got %d", count)
	}
	if descs != "updated by insert_on_update" {
		return fmt.Errorf("expected descs='updated by insert_on_update', got '%s'", descs)
	}

	fmt.Printf("data verified: count=%d, descs=%s\n", count, descs)
	return nil
}

func verifyUndoLogCleaned(ctx context.Context) error {
	var count int64
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM undo_log").Scan(&count)
	if err != nil {
		return fmt.Errorf("query undo_log failed: %w", err)
	}

	if count != 0 {
		return fmt.Errorf("undo_log not cleaned, count=%d", count)
	}

	fmt.Println("undo_log cleaned successfully")
	return nil
}

func initDB() {
	var err error
	db, err = sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true&interpolateParams=true")
	if err != nil {
		panic("init db error: " + err.Error())
	}
}
