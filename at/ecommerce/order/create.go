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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"seata.apache.org/seata-go-samples/util"
	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

type OrderRequest struct {
	UserID        string `json:"userId"`
	CommodityCode string `json:"commodityCode"`
	Count         int    `json:"count"`
	Money         int    `json:"money"`
}

type InventoryRequest struct {
	CommodityCode string `json:"commodityCode"`
	Count         int    `json:"count"`
}

type AccountRequest struct {
	UserID string `json:"userId"`
	Money  int    `json:"money"`
}

func createOrder(c *gin.Context) error {
	var req OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return util.NewValidationError(err.Error())
	}
	if strings.TrimSpace(req.UserID) == "" {
		return util.NewValidationError("userId is required")
	}
	if strings.TrimSpace(req.CommodityCode) == "" {
		return util.NewValidationError("commodityCode is required")
	}
	if req.Count <= 0 {
		return util.NewValidationError("count must be greater than 0")
	}
	if req.Money <= 0 {
		return util.NewValidationError("money must be greater than 0")
	}

	return tm.WithGlobalTx(c.Request.Context(), &tm.GtxConfig{
		Name:    "ATSampleEcommerceCreateOrder",
		Timeout: time.Second * 30,
	}, func(ctx context.Context) error {
		if err := insertOrder(ctx, req); err != nil {
			return err
		}
		if err := deductInventory(ctx, req); err != nil {
			return err
		}
		if err := deductAccount(ctx, req); err != nil {
			return err
		}
		return nil
	})
}

func insertOrder(ctx context.Context, req OrderRequest) error {
	query := "insert into order_tbl(user_id, commodity_code, count, money, status) values (?, ?, ?, ?, ?)"
	ret, err := db.ExecContext(ctx, query, req.UserID, req.CommodityCode, req.Count, req.Money, "CREATED")
	if err != nil {
		return err
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("create order affected unexpected rows: %d", rows)
	}
	return nil
}

func deductInventory(ctx context.Context, req OrderRequest) error {
	payload, err := json.Marshal(InventoryRequest{
		CommodityCode: req.CommodityCode,
		Count:         req.Count,
	})
	if err != nil {
		return err
	}

	log.Infof("call inventory service, xid=%s", tm.GetXID(ctx))
	return postJSON(ctx, inventoryService+"/deductInventory", payload)
}

func deductAccount(ctx context.Context, req OrderRequest) error {
	payload, err := json.Marshal(AccountRequest{
		UserID: req.UserID,
		Money:  req.Money,
	})
	if err != nil {
		return err
	}

	log.Infof("call account service, xid=%s", tm.GetXID(ctx))
	return postJSON(ctx, accountService+"/deductAccount", payload)
}

func postJSON(ctx context.Context, url string, payload []byte) error {
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(requestCtx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	httpReq.Header.Set(constant.XidKey, tm.GetXID(ctx))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return util.NewDownstreamError(0, fmt.Sprintf("request %s failed: %v", url, err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		message := strings.TrimSpace(string(body))
		var response util.APIResponse
		if err := json.Unmarshal(body, &response); err == nil {
			if response.Error != "" {
				message = response.Error
			} else if response.Message != "" {
				message = response.Message
			}
		}
		return util.NewDownstreamError(resp.StatusCode, message)
	}
	return nil
}
