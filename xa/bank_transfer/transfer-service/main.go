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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"seata.apache.org/seata-go/pkg/client"
	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/tm"
)

const defaultListenAddr = ":18080"

var httpClient = &http.Client{Timeout: 10 * time.Second}

type transferRequest struct {
	FromAccountNo string `json:"from_account_no" binding:"required"`
	ToAccountNo   string `json:"to_account_no" binding:"required"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
}

func main() {
	client.InitPath(resolveSeataConfig())

	r := gin.Default()
	r.POST("/transfer", func(c *gin.Context) {
		var req transferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		executeTransfer(c, req)
	})
	r.POST("/transfer/success", func(c *gin.Context) {
		executeTransfer(c, transferRequest{
			FromAccountNo: "A-1001",
			ToAccountNo:   "B-2001",
			Amount:        100,
		})
	})
	r.POST("/transfer/fail", func(c *gin.Context) {
		executeTransfer(c, transferRequest{
			FromAccountNo: "A-1001",
			ToAccountNo:   "B-FROZEN",
			Amount:        100,
		})
	})

	addr := getenv("TRANSFER_ADDR", defaultListenAddr)
	if err := r.Run(addr); err != nil {
		panic(fmt.Sprintf("start transfer-service failed: %v", err))
	}
}

func executeTransfer(c *gin.Context, req transferRequest) {
	err := tm.WithGlobalTx(c.Request.Context(), &tm.GtxConfig{
		Name:    "XABankTransfer",
		Timeout: 30 * time.Second,
	}, func(txCtx context.Context) error {
		if err := postJSON(txCtx, getenv("BANK_A_URL", "http://127.0.0.1:18081")+"/debit", map[string]any{
			"account_no": req.FromAccountNo,
			"amount":     req.Amount,
		}); err != nil {
			return err
		}

		if err := postJSON(txCtx, getenv("BANK_B_URL", "http://127.0.0.1:18082")+"/credit", map[string]any{
			"account_no": req.ToAccountNo,
			"amount":     req.Amount,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "committed"})
}

func postJSON(ctx context.Context, url string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constant.XidKey, tm.GetXID(ctx))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("post %s returned %s: %s", url, resp.Status, string(respBody))
	}
	return nil
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
