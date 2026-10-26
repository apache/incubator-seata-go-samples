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

	"seata.apache.org/seata-go-samples/util"
	"seata.apache.org/seata-go/v2/pkg/client"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

var db *sql.DB

func main() {
	client.InitPath("./conf/seatago.yml")
	db = util.GetXAMySqlDb()

	contextInstance := context.Background()
	isolatedRecordId := int64(2002)

	cleanupRecord(contextInstance, isolatedRecordId)
	prepareBaselineRecord(contextInstance, isolatedRecordId)

	expectedNewCount := int64(480)
	expectedNewDescription := "updated via xa select for update"

	transactionError := tm.WithGlobalTx(contextInstance, &tm.GtxConfig{
		Name:    "XATest_SelectForUpdate",
		Timeout: time.Second * 30,
	}, func(txCtx context.Context) error {
		selectStatement := "SELECT id, count FROM order_tbl WHERE id = ? FOR UPDATE"
		queryRows, queryError := db.QueryContext(txCtx, selectStatement, isolatedRecordId)
		if queryError != nil {
			return queryError
		}

		var currentId int64
		var currentCount int64
		hasData := false
		if queryRows.Next() {
			hasData = true
			if scanError := queryRows.Scan(&currentId, &currentCount); scanError != nil {
				_ = queryRows.Close()
				return scanError
			}
		}
		closeError := queryRows.Close()
		if closeError != nil {
			return closeError
		}
		if !hasData {
			return fmt.Errorf("target record id %d not found", isolatedRecordId)
		}

		updateStatement := "UPDATE order_tbl SET count = ?, descs = ? WHERE id = ?"
		executionResult, executionError := db.ExecContext(txCtx, updateStatement, expectedNewCount, expectedNewDescription, isolatedRecordId)
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
		panic(fmt.Sprintf("select for update transaction failed: %v", transactionError))
	}

	var recordedCount int64
	var recordedDescription string
	queryError := db.QueryRowContext(contextInstance,
		"SELECT count, descs FROM order_tbl WHERE id = ?", isolatedRecordId).Scan(&recordedCount, &recordedDescription)
	if queryError != nil {
		panic(fmt.Sprintf("query record verification failed: %v", queryError))
	}

	if recordedCount != expectedNewCount {
		panic(fmt.Sprintf("expected count %d, got %d", expectedNewCount, recordedCount))
	}
	if recordedDescription != expectedNewDescription {
		panic(fmt.Sprintf("expected descs '%s', got '%s'", expectedNewDescription, recordedDescription))
	}

	cleanupRecord(contextInstance, isolatedRecordId)
	fmt.Println("xa select_for_update integration test passed")
}

func prepareBaselineRecord(ctx context.Context, recordId int64) {
	statement := "INSERT INTO order_tbl (id, user_id, commodity_code, count, money, descs) VALUES (?, ?, ?, ?, ?, ?)"
	_, insertError := db.ExecContext(ctx, statement, recordId, "NO-XA-SELECT-LOCK-202", "C300004", 100, 10, "initial select desc")
	if insertError != nil {
		panic(fmt.Sprintf("prepare baseline record failed: %v", insertError))
	}
}

func cleanupRecord(ctx context.Context, recordId int64) {
	_, _ = db.ExecContext(ctx, "DELETE FROM order_tbl WHERE id = ?", recordId)
}
