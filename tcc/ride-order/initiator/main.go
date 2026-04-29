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
	"fmt"
	"net/http"
	"time"

	"github.com/parnurzeal/gorequest"

	"seata.apache.org/seata-go/pkg/client"
	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	orderServiceURL    = "http://127.0.0.1:8001"
	dispatchServiceURL = "http://127.0.0.1:8002"
	pricingServiceURL  = "http://127.0.0.1:8003"
	couponServiceURL   = "http://127.0.0.1:8004"
	capacityServiceURL = "http://127.0.0.1:8005"
)

func main() {
	client.InitPath("seatago.yml")
	ctx := context.Background()

	// ========== Scenario 1: All services succeed ==========
	fmt.Println("\n========== Scenario 1: All services succeed ==========")

	err := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "RideOrderSuccess",
		Timeout: 60 * time.Second,
	}, func(ctx context.Context) error {
		return rideOrderBusiness(ctx, false)
	})
	if err != nil {
		log.Errorf("Scenario 1 FAILED: %v", err)
	} else {
		log.Infof("Scenario 1 SUCCESS: ride order confirmed")
	}

	// Wait for Seata async Confirm callbacks
	time.Sleep(3 * time.Second)

	// ========== Scenario 2: Dispatch fails, triggers global rollback ==========
	fmt.Println("\n========== Scenario 2: Dispatch fails, triggers global rollback ==========")

	err = tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "RideOrderDispatchFail",
		Timeout: 60 * time.Second,
	}, func(ctx context.Context) error {
		return rideOrderBusiness(ctx, true)
	})
	if err != nil {
		log.Infof("Scenario 2 EXPECTED FAILURE: %v", err)
		log.Infof("All resources should be released via Cancel methods")
	} else {
		log.Errorf("Scenario 2 should have failed but succeeded")
	}

	// Wait for Cancel callbacks
	time.Sleep(3 * time.Second)
	fmt.Println("\n========== Done ==========")

	<-make(chan struct{})
}

func rideOrderBusiness(ctx context.Context, simulateDispatchFail bool) error {
	xid := tm.GetXID(ctx)

	// 1. Create pending order
	if err := callService(ctx, orderServiceURL, fmt.Sprintf(
		`{"passenger_id":"passenger-001"}`)); err != nil {
		return fmt.Errorf("order-service prepare failed: %v", err)
	}
	log.Infof("[Initiator] order-service prepared, xid=%s", xid)

	// 2. Lock estimated fare
	if err := callService(ctx, pricingServiceURL, fmt.Sprintf(
		`{"order_id":1,"amount":2800}`)); err != nil {
		return fmt.Errorf("pricing-service prepare failed: %v", err)
	}
	log.Infof("[Initiator] pricing-service prepared, xid=%s", xid)

	// 3. Freeze coupon
	couponID := 1
	if simulateDispatchFail {
		couponID = 2 // use coupon#2 because coupon#1 was consumed in scenario 1
	}
	if err := callService(ctx, couponServiceURL, fmt.Sprintf(
		`{"coupon_id":%d}`, couponID)); err != nil {
		return fmt.Errorf("coupon-service prepare failed: %v", err)
	}
	log.Infof("[Initiator] coupon-service prepared, xid=%s", xid)

	// 4. Reserve vehicle slot
	if err := callService(ctx, capacityServiceURL,
		`{"capacity_id":1}`); err != nil {
		return fmt.Errorf("capacity-service prepare failed: %v", err)
	}
	log.Infof("[Initiator] capacity-service prepared, xid=%s", xid)

	// 5. Reserve driver — THIS WILL FAIL in scenario 2
	if err := callService(ctx, dispatchServiceURL, fmt.Sprintf(
		`{"order_id":1,"simulate_fail":%t}`, simulateDispatchFail)); err != nil {
		return fmt.Errorf("dispatch-service prepare failed: %v", err)
	}
	log.Infof("[Initiator] dispatch-service prepared, xid=%s", xid)

	return nil
}

func callService(ctx context.Context, serviceURL string, jsonBody string) error {
	resp, body, errs := gorequest.New().
		Post(serviceURL+"/prepare").
		Set(constant.XidKey, tm.GetXID(ctx)).
		Type("json").
		Send(jsonBody).
		End()
	if len(errs) != 0 {
		return errs[0]
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service returned %d: %s", resp.StatusCode, body)
	}
	return nil
}
