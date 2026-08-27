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

type PricingRequest struct {
	OrderID int64 `json:"order_id"`
	Amount  int   `json:"amount"`
}

type PricingService struct{}

// pricingRecords tracks xid -> priceLockID for Commit/Rollback lookup.
var pricingRecords sync.Map

func (s *PricingService) GetActionName() string {
	return "PricingService"
}

// Prepare locks the estimated fare (Try phase).
func (s *PricingService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	req, ok := params.(PricingRequest)
	if !ok {
		return false, fmt.Errorf("invalid params type %T, want PricingRequest", params)
	}
	result, err := common.DB.ExecContext(ctx,
		"INSERT INTO price_locks (order_id, amount, status) VALUES (?, ?, 0)", req.OrderID, req.Amount)
	if err != nil {
		return false, fmt.Errorf("pricing prepare failed: %v", err)
	}
	lockID, _ := result.LastInsertId()
	xid := tm.GetXID(ctx)
	pricingRecords.Store(xid, lockID)
	log.Infof("[Pricing-Try] locked fare %d cents for order %d, lockID=%d, xid=%s", req.Amount, req.OrderID, lockID, xid)
	return true, nil
}

// Commit confirms the fare lock: locked -> confirmed. Idempotent via status check.
func (s *PricingService) Commit(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	lockID, ok := pricingRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Pricing-Confirm] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	_, err := common.DB.Exec("UPDATE price_locks SET status=1 WHERE id=? AND status=0", lockID)
	if err != nil {
		return false, fmt.Errorf("pricing confirm failed: %v", err)
	}
	pricingRecords.Delete(bac.Xid)
	log.Infof("[Pricing-Confirm] fare lock %d confirmed, xid=%s", lockID, bac.Xid)
	return true, nil
}

// Rollback releases the fare lock: locked -> released. Idempotent via status check.
func (s *PricingService) Rollback(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	lockID, ok := pricingRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Pricing-Cancel] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	_, err := common.DB.Exec("UPDATE price_locks SET status=2 WHERE id=? AND status=0", lockID)
	if err != nil {
		return false, fmt.Errorf("pricing cancel failed: %v", err)
	}
	pricingRecords.Delete(bac.Xid)
	log.Infof("[Pricing-Cancel] fare lock %d released, xid=%s", lockID, bac.Xid)
	return true, nil
}
