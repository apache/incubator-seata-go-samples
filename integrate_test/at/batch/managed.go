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
	"reflect"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	seataSQL "seata.apache.org/seata-go/v2/pkg/datasource/sql"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

var managedSuccessIDs = [3]int64{91001, 91002, 91003}

var managedSuccessBaseline = []orderRow{
	{ID: 91001, UserID: "batch-managed-1", CommodityCode: "batch", Count: 10, Money: 100, Descs: "managed baseline 1"},
	{ID: 91002, UserID: "batch-managed-2", CommodityCode: "batch", Count: 20, Money: 200, Descs: "managed baseline 2"},
	{ID: 91003, UserID: "batch-managed-3", CommodityCode: "batch", Count: 30, Money: 300, Descs: "managed baseline 3"},
}

type orderRow struct {
	ID            int64
	UserID        string
	CommodityCode string
	Count         int64
	Money         int64
	Descs         string
}

var managedFailureIDs = [3]int64{92001, 92002, 92003}

var managedFailureBaseline = []orderRow{
	{ID: 92002, UserID: "batch-existing", CommodityCode: "batch", Count: 20, Money: 200, Descs: "duplicate baseline"},
}

type rowQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (s batchSuite) runManagedSuccess(ctx context.Context) error {
	if err := s.restoreFixture(ctx, managedSuccessIDs, managedSuccessBaseline); err != nil {
		return fmt.Errorf("restore fixture: %w", err)
	}

	expected := append([]orderRow(nil), managedSuccessBaseline...)
	expected[0].Count, expected[0].Descs = 11, "managed updated 1"
	expected[1].Count, expected[1].Descs = 22, "managed updated 2"
	expected[2].Count, expected[2].Descs = 33, "managed updated 3"

	var xid string
	err := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "ATSemanticBatchManagedSuccess",
		Timeout: caseTimeout,
	}, func(txCtx context.Context) error {
		xid = tm.GetXID(txCtx)
		if xid == "" {
			return fmt.Errorf("global transaction xid is empty")
		}

		result, err := seataSQL.ExecBatchContext(txCtx, s.db,
			"UPDATE order_tbl SET count = ?, descs = ? WHERE id = ?",
			[][]any{
				{int64(11), "managed updated 1", int64(91001)},
				{int64(22), "managed updated 2", int64(91002)},
				{int64(33), "managed updated 3", int64(91003)},
			})
		if err != nil {
			return fmt.Errorf("execute batch: %w", err)
		}
		if err := assertSuccessfulBatch(result, seataSQL.BatchTransactionCommitted); err != nil {
			return err
		}
		if err := assertRows(txCtx, s.db, managedSuccessIDs, expected); err != nil {
			return fmt.Errorf("verify locally committed rows: %w", err)
		}
		if err := s.assertUndoLogCount(txCtx, xid, 1); err != nil {
			return fmt.Errorf("verify single branch undo log: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("global transaction did not commit: %w", err)
	}

	if err := assertRows(ctx, s.db, managedSuccessIDs, expected); err != nil {
		return fmt.Errorf("verify globally committed rows: %w", err)
	}
	if err := s.waitForUndoLogCleanup(ctx, xid); err != nil {
		return err
	}
	return nil
}

func (s batchSuite) runManagedPartialFailure(ctx context.Context) error {
	if err := s.restoreFixture(ctx, managedFailureIDs, managedFailureBaseline); err != nil {
		return fmt.Errorf("restore fixture: %w", err)
	}

	var xid string
	err := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "ATSemanticBatchManagedPartialFailure",
		Timeout: caseTimeout,
	}, func(txCtx context.Context) error {
		xid = tm.GetXID(txCtx)
		if xid == "" {
			return fmt.Errorf("global transaction xid is empty")
		}

		result, batchErr := seataSQL.ExecBatchContext(txCtx, s.db,
			"INSERT INTO order_tbl (id, user_id, commodity_code, count, money, descs) VALUES (?, ?, ?, ?, ?, ?)",
			[][]any{
				{int64(92001), "batch-inserted-1", "batch", int64(10), int64(100), "inserted before failure"},
				{int64(92002), "batch-duplicate", "batch", int64(20), int64(200), "deterministic duplicate"},
				{int64(92003), "batch-not-executed", "batch", int64(30), int64(300), "must not execute"},
			})
		if batchErr == nil {
			return fmt.Errorf("expected duplicate-key batch failure")
		}
		var mysqlErr *mysqlDriver.MySQLError
		if !errors.As(batchErr, &mysqlErr) || mysqlErr.Number != 1062 {
			return fmt.Errorf("expected MySQL duplicate-key error 1062, got %w", batchErr)
		}
		if err := assertPartialFailureBatch(result); err != nil {
			return err
		}
		if err := assertRows(txCtx, s.db, managedFailureIDs, managedFailureBaseline); err != nil {
			return fmt.Errorf("verify automatic local rollback: %w", err)
		}
		if err := s.assertUndoLogCount(txCtx, xid, 0); err != nil {
			return fmt.Errorf("verify failed local transaction did not register a branch: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("enclosing global transaction did not commit after handled batch failure: %w", err)
	}

	if err := assertRows(ctx, s.db, managedFailureIDs, managedFailureBaseline); err != nil {
		return fmt.Errorf("verify rows after enclosing global commit: %w", err)
	}
	return s.assertUndoLogCount(ctx, xid, 0)
}

func assertSuccessfulBatch(result seataSQL.BatchResult, transactionState seataSQL.BatchTransactionState) error {
	if len(result.Items) != 3 {
		return fmt.Errorf("expected 3 batch items, got %d", len(result.Items))
	}
	if result.Outcome.FailedIndex != seataSQL.NoFailedBatchItem {
		return fmt.Errorf("expected failed index %d, got %d", seataSQL.NoFailedBatchItem, result.Outcome.FailedIndex)
	}
	if result.Outcome.FailurePhase != seataSQL.BatchPhaseNone {
		return fmt.Errorf("expected no failure phase, got %d", result.Outcome.FailurePhase)
	}
	if result.Outcome.TransactionState != transactionState {
		return fmt.Errorf("expected transaction state %d, got %d", transactionState, result.Outcome.TransactionState)
	}
	for index, item := range result.Items {
		if item.Index != index {
			return fmt.Errorf("item %d reported index %d", index, item.Index)
		}
		if item.State != seataSQL.BatchItemExecuted {
			return fmt.Errorf("item %d expected executed state, got %d", index, item.State)
		}
		rowsAffected, err := item.RowsAffected()
		if err != nil {
			return fmt.Errorf("item %d rows affected: %w", index, err)
		}
		if rowsAffected != 1 {
			return fmt.Errorf("item %d expected 1 affected row, got %d", index, rowsAffected)
		}
	}
	return nil
}

func assertPartialFailureBatch(result seataSQL.BatchResult) error {
	if len(result.Items) != 3 {
		return fmt.Errorf("expected 3 batch items, got %d", len(result.Items))
	}
	if result.Outcome.FailedIndex != 1 {
		return fmt.Errorf("expected failed index 1, got %d", result.Outcome.FailedIndex)
	}
	if result.Outcome.FailurePhase != seataSQL.BatchPhaseExecute {
		return fmt.Errorf("expected execute failure phase, got %d", result.Outcome.FailurePhase)
	}
	if result.Outcome.TransactionState != seataSQL.BatchTransactionRolledBack {
		return fmt.Errorf("expected rolled-back transaction state, got %d", result.Outcome.TransactionState)
	}
	expectedStates := []seataSQL.BatchItemState{
		seataSQL.BatchItemExecuted,
		seataSQL.BatchItemFailed,
		seataSQL.BatchItemNotExecuted,
	}
	for index, item := range result.Items {
		if item.Index != index {
			return fmt.Errorf("item %d reported index %d", index, item.Index)
		}
		if item.State != expectedStates[index] {
			return fmt.Errorf("item %d expected state %d, got %d", index, expectedStates[index], item.State)
		}
	}
	if result.Items[1].Err() == nil {
		return fmt.Errorf("failed item did not preserve its execution error")
	}
	rowsAffected, err := result.Items[0].RowsAffected()
	if err != nil {
		return fmt.Errorf("first item rows affected: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("first item expected 1 affected row, got %d", rowsAffected)
	}
	return nil
}

func (s batchSuite) restoreFixture(ctx context.Context, ids [3]int64, expected []orderRow) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM order_tbl WHERE id IN (?, ?, ?)", ids[0], ids[1], ids[2]); err != nil {
		return err
	}
	for _, row := range expected {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO order_tbl (id, user_id, commodity_code, count, money, descs) VALUES (?, ?, ?, ?, ?, ?)",
			row.ID, row.UserID, row.CommodityCode, row.Count, row.Money, row.Descs); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func assertRows(ctx context.Context, queryer rowQueryer, ids [3]int64, expected []orderRow) error {
	rows, err := queryer.QueryContext(ctx,
		"SELECT id, user_id, commodity_code, count, money, descs FROM order_tbl WHERE id IN (?, ?, ?) ORDER BY id",
		ids[0], ids[1], ids[2])
	if err != nil {
		return err
	}
	defer rows.Close()

	actual := make([]orderRow, 0, len(expected))
	for rows.Next() {
		var row orderRow
		if err := rows.Scan(&row.ID, &row.UserID, &row.CommodityCode, &row.Count, &row.Money, &row.Descs); err != nil {
			return err
		}
		actual = append(actual, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("expected rows %+v, got %+v", expected, actual)
	}
	return nil
}

func (s batchSuite) assertUndoLogCount(ctx context.Context, xid string, expected int) error {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM undo_log WHERE xid = ?", xid).Scan(&count); err != nil {
		return err
	}
	if count != expected {
		return fmt.Errorf("expected %d undo-log rows for xid %s, got %d", expected, xid, count)
	}
	return nil
}

func (s batchSuite) waitForUndoLogCleanup(ctx context.Context, xid string) error {
	deadline := time.NewTimer(pollTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := s.assertUndoLogCount(ctx, xid, 0); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("undo log for xid %s was not cleaned within %s", xid, pollTimeout)
		case <-ticker.C:
		}
	}
}
