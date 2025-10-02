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

	"seata.apache.org/seata-go-samples/tcc/rocketmq/service"
	"seata.apache.org/seata-go/pkg/client"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

func main() {
	client.InitPath("../../../conf/seatago.yml")

	nameServers := []string{"127.0.0.1:9876"}
	if err := service.InitRocketMQProducer(nameServers, "SeataProducerGroup"); err != nil {
		log.Errorf("init rocketmq producer failed: %v", err)
		return
	}

	_ = tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name: "TccSampleRocketMQGlobalTx",
	}, business)

	<-make(chan struct{})
}

func business(ctx context.Context) (re error) {
	log.Infof("=== Starting global transaction business logic ===")

	log.Infof("Step 1: Calling OrderService.Prepare")
	if _, re = service.NewOrderServiceProxy().Prepare(ctx, "order-001"); re != nil {
		log.Errorf("OrderService prepare error, %v", re)
		return
	}
	log.Infof("Step 1: OrderService.Prepare completed successfully")

	log.Infof("Step 2: Sending RocketMQ message")
	if re = service.SendMessage(ctx, "test-topic", "test-tag", "Hello Seata RocketMQ"); re != nil {
		log.Errorf("SendMessage error, %v", re)
		return
	}
	log.Infof("Step 2: RocketMQ message sent successfully")

	log.Infof("=== Business logic completed, waiting for global commit ===")
	return
}
