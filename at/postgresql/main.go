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
	"log"
	"os"
	"time"

	"seata.apache.org/seata-go/v2/pkg/client"
	seataSQL "seata.apache.org/seata-go/v2/pkg/datasource/sql"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

const (
	defaultDSN = "postgres://seata:seata@127.0.0.1:5432/seata_at?sslmode=disable"
	userID     = "U1001"
	accountID  = 1
	orderMoney = 100
)

var errSimulated = errors.New("simulated failure after the local changes")

func main() {
	client.InitPath(resolveSeataConfig())

	db, err := sql.Open(seataSQL.SeataATPostgresDriver, getenv("POSTGRES_DSN", defaultDSN))
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	printState(ctx, db, "before")

	// The global transaction returns nil, so Seata commits it.
	if err := placeOrder(ctx, db, false); err != nil {
		log.Fatalf("commit case failed: %v", err)
	}
	printState(ctx, db, "after commit")

	// Seata finishes a commit in the background by deleting the undo log,
	// so wait for that before going on.
	waitUndoLogCleared(ctx, db)

	// The global transaction returns an error, so Seata rolls it back
	// with the undo log written by the AT driver.
	balance, orders := readState(ctx, db)
	err = placeOrder(ctx, db, true)
	if !errors.Is(err, errSimulated) {
		log.Fatalf("rollback case: expected the simulated error, got: %v", err)
	}
	printState(ctx, db, "after rollback")

	newBalance, newOrders := readState(ctx, db)
	if newBalance != balance || newOrders != orders {
		log.Fatalf("rollback did not restore the data: balance %d -> %d, orders %d -> %d",
			balance, newBalance, orders, newOrders)
	}
	log.Println("rollback restored the data")
}

// placeOrder takes money from the account and creates an order in one global
// transaction. When fail is true it returns an error after both statements ran.
func placeOrder(ctx context.Context, db *sql.DB, fail bool) error {
	return tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "ATPostgresSample",
		Timeout: 30 * time.Second,
	}, func(ctx context.Context) error {
		log.Printf("global transaction started, xid=%s", tm.GetXID(ctx))

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE account_tbl SET balance = balance - $1 WHERE id = $2",
			orderMoney, accountID); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("update account: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO order_tbl (user_id, commodity_code, count, money) VALUES ($1, $2, $3, $4)",
			userID, "C100", 1, orderMoney); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert order: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}

		if fail {
			return errSimulated
		}
		return nil
	})
}

func waitUndoLogCleared(ctx context.Context, db *sql.DB) {
	deadline := time.Now().Add(30 * time.Second)
	for {
		var n int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM undo_log").Scan(&n); err != nil {
			log.Fatalf("count undo log: %v", err)
		}
		if n == 0 {
			log.Println("undo log cleaned up after commit")
			return
		}
		if time.Now().After(deadline) {
			log.Fatalf("undo log still has %d rows after commit", n)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func readState(ctx context.Context, db *sql.DB) (balance, orders int64) {
	if err := db.QueryRowContext(ctx, "SELECT balance FROM account_tbl WHERE id = $1", accountID).Scan(&balance); err != nil {
		log.Fatalf("read balance: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM order_tbl WHERE user_id = $1", userID).Scan(&orders); err != nil {
		log.Fatalf("count orders: %v", err)
	}
	return balance, orders
}

func printState(ctx context.Context, db *sql.DB, label string) {
	balance, orders := readState(ctx, db)
	log.Printf("%s: balance=%d orders=%d", label, balance, orders)
}

func resolveSeataConfig() string {
	if path := os.Getenv("SEATA_CONFIG"); path != "" {
		return path
	}
	for _, path := range []string{"./conf/seatago.yml", "../../conf/seatago.yml"} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "./conf/seatago.yml"
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
