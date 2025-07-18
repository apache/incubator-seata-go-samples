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

	_ "github.com/go-sql-driver/mysql"
	"seata.apache.org/seata-go/pkg/client"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"

	"seata.apache.org/seata-go-samples/tcc/fence/service"
)

func main() {
	client.InitPath("../../../conf/seatago.yml")
	_ = tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name: "TccSampleLocalGlobalTx",
	}, business)
	<-make(chan struct{})
}

func business(ctx context.Context) (re error) {
	tccService := service.NewTestTCCServiceBusinessProxy()
	tccService2 := service.NewTestTCCServiceBusiness2Proxy()
	_, re = tccService.Prepare(ctx, 1)
	if re != nil {
		log.Errorf("TestTCCServiceBusiness prepare error, %v", re.Error())
		return
	}
	_, re = tccService2.Prepare(ctx, 3)
	if re != nil {
		log.Errorf("TestTCCServiceBusiness2 prepare error, %v", re.Error())
		return
	}

	return
}
