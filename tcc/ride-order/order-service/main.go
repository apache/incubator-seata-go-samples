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
	"net/http"

	"github.com/gin-gonic/gin"

	"seata.apache.org/seata-go/pkg/client"
	ginmiddleware "seata.apache.org/seata-go/pkg/integration/gin"
	"seata.apache.org/seata-go/pkg/rm/tcc"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"

	"seata.apache.org/seata-go-samples/tcc/ride-order/common"
)

func main() {
	client.InitPath("seatago.yml")
	common.InitDB()

	r := gin.Default()
	r.Use(ginmiddleware.TransactionMiddleware())

	proxy, err := tcc.NewTCCServiceProxy(&OrderService{})
	if err != nil {
		log.Fatalf("create OrderService proxy error: %v", err)
	}

	r.POST("/prepare", func(c *gin.Context) {
		var req OrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx := c.Request.Context()
		if _, err := proxy.Prepare(ctx, req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Return the generated order id so the initiator can thread it through
		// the downstream service calls (pricing, dispatch) instead of hardcoding it.
		var orderID int64
		if v, ok := orderRecords.Load(tm.GetXID(ctx)); ok {
			orderID, _ = v.(int64)
		}
		c.JSON(http.StatusOK, gin.H{"message": "prepare ok", "order_id": orderID})
	})

	if err := r.Run(":8001"); err != nil {
		log.Fatalf("start order-service error: %v", err)
	}
}
