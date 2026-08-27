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
	"sync"

	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"

	"seata.apache.org/seata-go-samples/tcc/ride-order/common"
)

type OrderRequest struct {
	PassengerID string `json:"passenger_id"`
}

type OrderService struct{}

// orderRecords tracks xid -> orderID for Commit/Rollback lookup.
var orderRecords sync.Map

func (s *OrderService) GetActionName() string {
	return "OrderService"
}

// Prepare creates a pending ride order (Try phase).
func (s *OrderService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	req, ok := params.(OrderRequest)
	if !ok {
		return false, fmt.Errorf("invalid params type %T, want OrderRequest", params)
	}
	result, err := common.DB.ExecContext(ctx,
		"INSERT INTO ride_orders (passenger_id, status) VALUES (?, 0)", req.PassengerID)
	if err != nil {
		return false, fmt.Errorf("order prepare failed: %v", err)
	}
	orderID, _ := result.LastInsertId()
	xid := tm.GetXID(ctx)
	orderRecords.Store(xid, orderID)
	log.Infof("[Order-Try] created pending order %d for passenger %s, xid=%s", orderID, req.PassengerID, xid)
	return true, nil
}

// Commit confirms the order: pending -> confirmed. Idempotent via status check.
func (s *OrderService) Commit(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	orderID, ok := orderRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Order-Confirm] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	_, err := common.DB.Exec("UPDATE ride_orders SET status=1 WHERE id=? AND status=0", orderID)
	if err != nil {
		return false, fmt.Errorf("order confirm failed: %v", err)
	}
	orderRecords.Delete(bac.Xid)
	log.Infof("[Order-Confirm] order %d confirmed, xid=%s", orderID, bac.Xid)
	return true, nil
}

// Rollback cancels the order: pending -> canceled. Idempotent via status check.
func (s *OrderService) Rollback(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	orderID, ok := orderRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Order-Cancel] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	_, err := common.DB.Exec("UPDATE ride_orders SET status=2 WHERE id=? AND status=0", orderID)
	if err != nil {
		return false, fmt.Errorf("order cancel failed: %v", err)
	}
	orderRecords.Delete(bac.Xid)
	log.Infof("[Order-Cancel] order %d canceled, xid=%s", orderID, bac.Xid)
	return true, nil
}
