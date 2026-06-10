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
	"seata.apache.org/seata-go-samples/util"
)

type AccountRequest struct {
	UserID string `json:"userId"`
	Money  int    `json:"money"`
}

func deductAccount(c *gin.Context) error {
	var req AccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return util.NewValidationError(err.Error())
	}
	if strings.TrimSpace(req.UserID) == "" {
		return util.NewValidationError("userId is required")
	}
	if req.Money <= 0 {
		return util.NewValidationError("money must be greater than 0")
	}

	query := "update account_tbl set balance = balance - ? where user_id = ? and balance >= ?"
	ret, err := db.ExecContext(c.Request.Context(), query, req.Money, req.UserID, req.Money)
	if err != nil {
		return err
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return util.NewConflictError(fmt.Sprintf("insufficient balance for userId %s", req.UserID))
	}
	return nil
}
