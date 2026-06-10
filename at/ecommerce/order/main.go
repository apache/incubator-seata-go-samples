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
	"database/sql"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"seata.apache.org/seata-go-samples/util"
	"seata.apache.org/seata-go/pkg/client"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	db               *sql.DB
	inventoryService = "http://127.0.0.1:18081"
	accountService   = "http://127.0.0.1:18082"
)

func main() {
	client.InitPath("conf/seatago.yml")
	if err := util.SetDefaultEnv("MYSQL_DB", "seata_ecommerce_order"); err != nil {
		log.Fatalf("set MYSQL_DB default error: %v", err)
	}
	if value := os.Getenv("INVENTORY_SERVICE_URL"); value != "" {
		inventoryService = value
	}
	if value := os.Getenv("ACCOUNT_SERVICE_URL"); value != "" {
		accountService = value
	}
	db = util.GetAtMySqlDb()

	r := gin.Default()
	r.POST("/createOrder", createOrderHandler)

	if err := r.Run(":18080"); err != nil {
		log.Fatalf("start order service fatal: %v", err)
	}
}

func createOrderHandler(c *gin.Context) {
	log.Infof("receive create order request")
	if err := createOrder(c); err != nil {
		c.JSON(util.StatusCodeForError(err), util.APIResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, util.APIResponse{Message: "create order ok"})
}
