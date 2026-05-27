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
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/parnurzeal/gorequest"
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
		return err
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
	sql := "insert into order_tbl(user_id, commodity_code, count, money, status) values (?, ?, ?, ?, ?)"
	ret, err := db.ExecContext(ctx, sql, req.UserID, req.CommodityCode, req.Count, req.Money, "CREATED")
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

func deductInventory(ctx context.Context, req OrderRequest) (re error) {
	request := gorequest.New()
	payload, err := json.Marshal(InventoryRequest{
		CommodityCode: req.CommodityCode,
		Count:         req.Count,
	})
	if err != nil {
		return err
	}

	log.Infof("call inventory service, xid=%s", tm.GetXID(ctx))
	request.Post(inventoryService + "/deductInventory").
		Set(constant.XidKey, tm.GetXID(ctx)).
		Send(string(payload)).
		Set("Content-Type", "application/json").
		End(func(response gorequest.Response, body string, errs []error) {
			if len(errs) > 0 {
				re = errs[0]
				return
			}
			if response == nil || response.StatusCode != http.StatusOK {
				re = fmt.Errorf("deduct inventory failed: %s", body)
			}
		})
	return
}

func deductAccount(ctx context.Context, req OrderRequest) (re error) {
	request := gorequest.New()
	payload, err := json.Marshal(AccountRequest{
		UserID: req.UserID,
		Money:  req.Money,
	})
	if err != nil {
		return err
	}

	log.Infof("call account service, xid=%s", tm.GetXID(ctx))
	request.Post(accountService + "/deductAccount").
		Set(constant.XidKey, tm.GetXID(ctx)).
		Send(string(payload)).
		Set("Content-Type", "application/json").
		End(func(response gorequest.Response, body string, errs []error) {
			if len(errs) > 0 {
				re = errs[0]
				return
			}
			if response == nil || response.StatusCode != http.StatusOK {
				re = fmt.Errorf("deduct account failed: %s", body)
			}
		})
	return
}
