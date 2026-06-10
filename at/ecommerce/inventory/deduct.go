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
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type InventoryRequest struct {
	CommodityCode string `json:"commodityCode"`
	Count         int    `json:"count"`
}

func deductInventory(c *gin.Context) error {
	var req InventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return err
	}
	if strings.TrimSpace(req.CommodityCode) == "" {
		return fmt.Errorf("commodityCode is required")
	}
	if req.Count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}

	sql := "update inventory_tbl set stock = stock - ? where commodity_code = ? and stock >= ?"
	ret, err := db.ExecContext(c.Request.Context(), sql, req.Count, req.CommodityCode, req.Count)
	if err != nil {
		return err
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("inventory not enough")
	}
	return nil
}
