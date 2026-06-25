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
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"seata.apache.org/seata-go/pkg/client"
	sql2 "seata.apache.org/seata-go/pkg/datasource/sql"
	ginmiddleware "seata.apache.org/seata-go/pkg/integration/gin"
)

const defaultListenAddr = ":18082"

var db *sql.DB

type creditRequest struct {
	AccountNo string `json:"account_no" binding:"required"`
	Amount    int64  `json:"amount" binding:"required,gt=0"`
}

type account struct {
	AccountNo string `json:"account_no"`
	Balance   int64  `json:"balance"`
	Frozen    bool   `json:"frozen"`
}

func main() {
	client.InitPath(resolveSeataConfig())
	db = openXADB()
	defer db.Close()

	r := gin.Default()
	r.ContextWithFallback = true
	// Read-only routes registered before Use() skip TransactionMiddleware.
	r.GET("/accounts/:accountNo", accountHandler)
	// Credit participates in the global XA branch; middleware injects the global XID into request context.
	r.Use(ginmiddleware.TransactionMiddleware())
	r.POST("/credit", creditHandler)

	addr := getenv("BANK_B_ADDR", defaultListenAddr)
	if err := r.Run(addr); err != nil {
		panic(fmt.Sprintf("start bank-b-service failed: %v", err))
	}
}

func creditHandler(c *gin.Context) {
	var req creditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := credit(c.Request.Context(), req.AccountNo, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "credited"})
}

func accountHandler(c *gin.Context) {
	acc, err := getAccount(c.Request.Context(), c.Param("accountNo"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, acc)
}

func credit(ctx context.Context, accountNo string, amount int64) error {
	ret, err := db.ExecContext(ctx,
		"update account_tbl set balance = balance + ? where account_no = ? and frozen = 0",
		amount, accountNo,
	)
	if err != nil {
		return fmt.Errorf("credit account %s failed: %w", accountNo, err)
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		return fmt.Errorf("read credit affected rows failed: %w", err)
	}
	if rows == 0 {
		acc, lookupErr := getAccount(context.Background(), accountNo)
		if lookupErr != nil {
			return lookupErr
		}
		if acc.Frozen {
			return fmt.Errorf("account %s is frozen and rejects credit", accountNo)
		}
		return fmt.Errorf("account %s did not accept credit", accountNo)
	}
	return nil
}

func getAccount(ctx context.Context, accountNo string) (*account, error) {
	var acc account
	err := db.QueryRowContext(ctx, "select account_no, balance, frozen from account_tbl where account_no = ?", accountNo).
		Scan(&acc.AccountNo, &acc.Balance, &acc.Frozen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("account %s not found", accountNo)
	}
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func openXADB() *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?multiStatements=true&interpolateParams=true",
		getenv("BANK_B_MYSQL_USERNAME", "root"),
		getenv("BANK_B_MYSQL_PASSWORD", "123456"),
		getenv("BANK_B_MYSQL_HOST", "127.0.0.1"),
		getenv("BANK_B_MYSQL_PORT", "3308"),
		getenv("BANK_B_MYSQL_DB", "db_bank_b"),
	)
	db, err := sql.Open(sql2.SeataXAMySQLDriver, dsn)
	if err != nil {
		panic(fmt.Sprintf("open bank-b XA datasource failed: %v", err))
	}
	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("ping bank-b datasource failed: %v", err))
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)
	return db
}

func resolveSeataConfig() string {
	if path := os.Getenv("SEATA_CONFIG"); path != "" {
		return path
	}
	for _, path := range []string{"./conf/seatago.yml", "../../../conf/seatago.yml"} {
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
