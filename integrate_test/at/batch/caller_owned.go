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
	"errors"
	"fmt"
	"time"

	seataSQL "seata.apache.org/seata-go/v2/pkg/datasource/sql"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

var callerOwnedIDs = [3]int64{93001, 93002, 93003}

var callerOwnedBaseline = []orderRow{
	{ID: 93001, UserID: "batch-caller-1", CommodityCode: "batch", Count: 10, Money: 100, Descs: "caller baseline 1"},
	{ID: 93002, UserID: "batch-caller-2", CommodityCode: "batch", Count: 20, Money: 200, Descs: "caller baseline 2"},
	{ID: 93003, UserID: "batch-caller-3", CommodityCode: "batch", Count: 30, Money: 300, Descs: "caller baseline 3"},
}

var triggerGlobalRollback = errors.New("trigger expected global rollback")

func (s batchSuite) runCallerOwnedGlobalRollback(ctx context.Context) error {
	if err := s.restoreFixture(ctx, callerOwnedIDs, callerOwnedBaseline); err != nil {
		return fmt.Errorf("restore fixture: %w", err)
	}

	var xid string
	globalErr := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "ATSemanticBatchCallerOwnedGlobalRollback",
		Timeout: caseTimeout,
	}, func(txCtx context.Context) error {
		xid = tm.GetXID(txCtx)
		if xid == "" {
			return fmt.Errorf("global transaction xid is empty")
		}

		tx, err := s.db.BeginTx(txCtx, nil)
		if err != nil {
			return fmt.Errorf("begin caller-owned transaction: %w", err)
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		result, err := seataSQL.ExecBatchInTxContext(txCtx, tx,
			"DELETE FROM order_tbl WHERE id = ?",
			[][]any{{int64(93001)}, {int64(93002)}, {int64(93003)}})
		if err != nil {
			return fmt.Errorf("execute batch: %w", err)
		}
		if err := assertSuccessfulBatch(result, seataSQL.BatchTransactionPending); err != nil {
			return err
		}
		if err := assertRows(txCtx, tx, callerOwnedIDs, []orderRow{}); err != nil {
			return fmt.Errorf("same transaction was not usable after batch: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("caller commit: %w", err)
		}
		committed = true

		if err := assertRows(txCtx, s.db, callerOwnedIDs, []orderRow{}); err != nil {
			return fmt.Errorf("verify locally committed deletes: %w", err)
		}
		if err := s.assertUndoLogCount(txCtx, xid, 1); err != nil {
			return fmt.Errorf("verify single branch undo log: %w", err)
		}
		return triggerGlobalRollback
	})
	if globalErr == nil {
		return fmt.Errorf("expected global rollback trigger error")
	}
	if !errors.Is(globalErr, triggerGlobalRollback) {
		return fmt.Errorf("global rollback failed: %w", globalErr)
	}

	if err := waitForRows(ctx, s.db, callerOwnedIDs, callerOwnedBaseline); err != nil {
		return fmt.Errorf("verify rows restored by global rollback: %w", err)
	}
	if err := s.waitForUndoLogCleanup(ctx, xid); err != nil {
		return err
	}
	return nil
}

func waitForRows(ctx context.Context, queryer rowQueryer, ids [3]int64, expected []orderRow) error {
	deadline := time.NewTimer(pollTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		lastErr = assertRows(ctx, queryer, ids, expected)
		if lastErr == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("rows did not reach expected state within %s: %w", pollTimeout, lastErr)
		case <-ticker.C:
		}
	}
}
