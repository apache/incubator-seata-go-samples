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

	"github.com/gin-gonic/gin"
	"seata.apache.org/seata-go-samples/util"
	"seata.apache.org/seata-go/pkg/client"
	ginmiddleware "seata.apache.org/seata-go/pkg/integration/gin"
	"seata.apache.org/seata-go/pkg/util/log"
)

var db *sql.DB

func main() {
	client.InitPath("conf/seatago.yml")
	if err := util.SetDefaultEnv("MYSQL_DB", "seata_ecommerce_account"); err != nil {
		log.Fatalf("set MYSQL_DB default error: %v", err)
	}
	db = util.GetAtMySqlDb()

	r := gin.Default()
	r.ContextWithFallback = true
	r.Use(ginmiddleware.TransactionMiddleware())
	r.POST("/deductAccount", deductAccountHandler)

	if err := r.Run(":18082"); err != nil {
		log.Fatalf("start account service fatal: %v", err)
	}
}

func deductAccountHandler(c *gin.Context) {
	log.Infof("receive deduct account request")
	if err := deductAccount(c); err != nil {
		log.Errorf("deduct account failed: %v", err)
		c.JSON(util.StatusCodeForError(err), util.APIResponse{Error: util.PublicErrorMessage(err)})
		return
	}
	c.JSON(http.StatusOK, util.APIResponse{Message: "deduct account ok"})
}
