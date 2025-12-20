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
	"time"

	"github.com/gin-gonic/gin"
	"seata.apache.org/seata-go-samples/quick_start/order/handler"
	"seata.apache.org/seata-go-samples/quick_start/order/service"
	"seata.apache.org/seata-go/pkg/tm"
)

func newService() *service.OrderService {
	return service.NewOrderService(gormDB, &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx",
		Timeout: 100 * time.Second,
	})
}

func main() {
	initConfig()
	web := handler.NewOrderHandler(newService())
	engine := gin.Default()
	web.Route(engine)
	engine.Run(":8080")
}
