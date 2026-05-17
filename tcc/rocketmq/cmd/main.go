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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"seata.apache.org/seata-go/v2/pkg/client"
	"seata.apache.org/seata-go/v2/pkg/integration/rocketmq"
	"seata.apache.org/seata-go/v2/pkg/tm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

var (
	mode = flag.String("mode", "commit", "Transaction mode: commit or rollback")
)

func main() {
	flag.Parse()

	// Initialize Seata client
	client.InitPath("../../../conf/seatago.yml")

	// Create RocketMQ producer
	// Note: SeataMQProducer internally creates TCC proxy - no need to wrap it again
	cfg := rocketmq.NewDefaultSeataMQProducerConfig()
	cfg.NameServerAddrs = []string{"127.0.0.1:9876"}
	cfg.GroupName = "seata-tcc-producer-group"
	cfg.InstanceName = "seata-tcc-producer-instance"

	producer, err := rocketmq.NewSeataMQProducer(cfg)
	if err != nil {
		log.Errorf("Create producer failed: %v", err)
		os.Exit(1)
	}

	if err := producer.Start(); err != nil {
		log.Errorf("Start producer failed: %v", err)
		os.Exit(1)
	}
	defer producer.Shutdown()

	// Execute global transaction
	err = tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "RocketMQTCCSample",
		Timeout: 60000,
	}, func(ctx context.Context) error {
		return sendMessage(ctx, producer, *mode)
	})

	if err != nil {
		log.Errorf("Global transaction failed: %v", err)
		os.Exit(1)
	}

	log.Infof("Global transaction completed successfully in %s mode", *mode)
}

func sendMessage(ctx context.Context, producer *rocketmq.SeataMQProducer, mode string) error {
	xid := tm.GetXID(ctx)
	log.Infof("Sending message in %s mode, XID: %s", mode, xid)

	// Prepare message payload
	payload, _ := json.Marshal(map[string]interface{}{
		"mode":      mode,
		"timestamp": time.Now().Unix(),
		"message":   "RocketMQ TCC test message",
		"xid":       xid,
	})

	msg := primitive.NewMessage("seata-tcc-test", payload)
	msg.WithTag("TCC_TEST")

	// SDK auto-detects global transaction context and applies TCC automatically
	// When tm.IsGlobalTx(ctx) is true, producer.Send() internally calls tccProxy.Prepare()
	result, err := producer.Send(ctx, msg)
	if err != nil {
		log.Errorf("Send message failed: %v", err)
		return err
	}

	log.Infof("Message sent successfully, msgId=%s, offsetMsgId=%s", result.MsgID, result.OffsetMsgID)

	// Simulate rollback scenario by returning error
	if mode == "rollback" {
		log.Infof("Simulating rollback scenario")
		return fmt.Errorf("simulated rollback scenario")
	}

	return nil
}
