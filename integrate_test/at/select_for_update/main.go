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
var expectedCount int64

func main() {
	client.InitPath("./conf/seatago.yml")
	initDB()

	ctx := context.Background()

	// execute select for update + update within global transaction
	err := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx_SelectForUpdate",
		Timeout: time.Second * 30,
	}, selectForUpdateAndModify)
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

	fmt.Println("select_for_update test passed")
}

// selectForUpdateAndModify tests SELECT ... FOR UPDATE followed by UPDATE
// This is the typical use case: lock the row first, then modify it
func selectForUpdateAndModify(ctx context.Context) error {
	// step 1: select for update to lock the row
	var id int64
	var count int64
	err := db.QueryRowContext(ctx,
		"SELECT id, count FROM order_tbl WHERE id = ? FOR UPDATE", 1).Scan(&id, &count)
	if err != nil {
		return fmt.Errorf("select for update failed: %w", err)
	}
	fmt.Printf("locked row: id=%d, count=%d\n", id, count)

	// step 2: update the locked row
	newCount := count + 50
	expectedCount = newCount // save for verification
	result, err := db.ExecContext(ctx,
		"UPDATE order_tbl SET count = ?, descs = ? WHERE id = ?",
		newCount, "updated by select_for_update", id)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected failed: %w", err)
	}
	fmt.Printf("update affected rows: %d\n", rows)

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

	if count != expectedCount {
		return fmt.Errorf("expected count=%d, got %d", expectedCount, count)
	}
	if descs != "updated by select_for_update" {
		return fmt.Errorf("expected descs='updated by select_for_update', got '%s'", descs)
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
