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
	"seata.apache.org/seata-go/pkg/util/log"

	"seata.apache.org/seata-go-samples/tcc/ride-order/common"
)

func main() {
	client.InitPath("../../../conf/seatago.yml")
	common.InitDB()

	r := gin.Default()
	r.Use(ginmiddleware.TransactionMiddleware())

	proxy, err := tcc.NewTCCServiceProxy(&DispatchService{})
	if err != nil {
		log.Fatalf("create DispatchService proxy error: %v", err)
	}

	r.POST("/prepare", func(c *gin.Context) {
		var req DispatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := proxy.Prepare(c, req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "prepare ok"})
	})

	if err := r.Run(":8002"); err != nil {
		log.Fatalf("start dispatch-service error: %v", err)
	}
}
