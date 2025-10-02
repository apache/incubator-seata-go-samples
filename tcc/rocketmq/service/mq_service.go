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

package service

import (
	"context"
	"fmt"

	"github.com/apache/rocketmq-client-go/v2/primitive"

	"seata.apache.org/seata-go/pkg/integration/rocketmq"
	"seata.apache.org/seata-go/pkg/util/log"
)

var seataMQProducer *rocketmq.SeataMQProducer

func InitRocketMQProducer(nameServers []string, groupName string) error {
	checker := rocketmq.NewTransactionChecker(&rocketmq.DefaultGlobalStatusChecker{})

	producer, err := rocketmq.NewRocketMQProducer(nameServers, groupName, checker)
	if err != nil {
		return fmt.Errorf("create rocketmq producer failed: %w", err)
	}

	seataMQProducer, err = rocketmq.CreateSeataMQProducer(producer)
	if err != nil {
		return fmt.Errorf("create seata mq producer failed: %w", err)
	}

	log.Infof("RocketMQ producer initialized successfully")
	return nil
}

func SendMessage(ctx context.Context, topic, tag, body string) error {
	msg := primitive.NewMessage(topic, []byte(body))
	msg.WithTag(tag)

	result, err := seataMQProducer.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("send message failed: %w", err)
	}

	log.Infof("Message sent successfully, result: %v", result)
	return nil
}
