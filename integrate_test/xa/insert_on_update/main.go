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
	isolatedRecordId := int64(2001)

	cleanupRecord(contextInstance, isolatedRecordId)
	prepareBaselineRecord(contextInstance, isolatedRecordId)

	targetCount := int64(350)
	targetDescription := "updated by xa insert on update"

	transactionError := tm.WithGlobalTx(contextInstance, &tm.GtxConfig{
		Name:    "XATest_InsertOnUpdate",
		Timeout: time.Second * 30,
	}, func(txCtx context.Context) error {
		statement := "INSERT INTO order_tbl (id, user_id, commodity_code, count, money, descs) " +
			"VALUES (?, ?, ?, ?, ?, ?) " +
			"ON DUPLICATE KEY UPDATE count = ?, descs = ?"

		executionResult, executionError := db.ExecContext(txCtx, statement,
			isolatedRecordId, "NO-XA-ON-UPDATE-201", "C300003", targetCount, 10, "temporary desc",
			targetCount, targetDescription)
		if executionError != nil {
			return executionError
		}

		rows, rowsError := executionResult.RowsAffected()
		if rowsError != nil {
			return rowsError
		}
		if rows == 0 {
			return fmt.Errorf("expected affected rows greater than 0, got 0")
		}
		return nil
	})

	if transactionError != nil {
		panic(fmt.Sprintf("insert on update transaction failed: %v", transactionError))
	}

	var recordedCount int64
	var recordedDescription string
	queryError := db.QueryRowContext(contextInstance,
		"SELECT count, descs FROM order_tbl WHERE id = ?", isolatedRecordId).Scan(&recordedCount, &recordedDescription)
	if queryError != nil {
		panic(fmt.Sprintf("query updated record failed: %v", queryError))
	}

	if recordedCount != targetCount {
		panic(fmt.Sprintf("expected count %d, got %d", targetCount, recordedCount))
	}
	if recordedDescription != targetDescription {
		panic(fmt.Sprintf("expected descs '%s', got '%s'", targetDescription, recordedDescription))
	}

	cleanupRecord(contextInstance, isolatedRecordId)
	fmt.Println("xa insert_on_update integration test passed")
}

func prepareBaselineRecord(ctx context.Context, recordId int64) {
	statement := "INSERT INTO order_tbl (id, user_id, commodity_code, count, money, descs) VALUES (?, ?, ?, ?, ?, ?)"
	_, insertError := db.ExecContext(ctx, statement, recordId, "NO-XA-ON-UPDATE-201", "C300003", 100, 10, "initial desc")
	if insertError != nil {
		panic(fmt.Sprintf("prepare baseline record failed: %v", insertError))
	}
}

func cleanupRecord(ctx context.Context, recordId int64) {
	_, _ = db.ExecContext(ctx, "DELETE FROM order_tbl WHERE id = ?", recordId)
}
