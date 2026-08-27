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

type CouponRequest struct {
	PassengerID string `json:"passenger_id"`
}

type CouponService struct{}

// couponRecords tracks xid -> couponID for Commit/Rollback lookup.
var couponRecords sync.Map

func (s *CouponService) GetActionName() string {
	return "CouponService"
}

// Prepare freezes an available coupon for the passenger: available -> frozen (Try phase).
// The coupon is selected dynamically so the sample stays re-runnable (no hardcoded id
// that gets permanently consumed after the first Confirm).
func (s *CouponService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	req, ok := params.(CouponRequest)
	if !ok {
		return false, fmt.Errorf("invalid params type %T, want CouponRequest", params)
	}

	var couponID int64
	err := common.DB.QueryRowContext(ctx,
		"SELECT id FROM coupons WHERE user_id=? AND status=0 ORDER BY id LIMIT 1", req.PassengerID).Scan(&couponID)
	if err != nil {
		return false, fmt.Errorf("no available coupon for %s: %v", req.PassengerID, err)
	}
	// Guard against a concurrent transaction grabbing the same coupon.
	result, err := common.DB.ExecContext(ctx,
		"UPDATE coupons SET status=1 WHERE id=? AND status=0", couponID)
	if err != nil {
		return false, fmt.Errorf("coupon prepare failed: %v", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return false, fmt.Errorf("coupon %d no longer available", couponID)
	}
	xid := tm.GetXID(ctx)
	couponRecords.Store(xid, couponID)
	log.Infof("[Coupon-Try] froze coupon %d for %s, xid=%s", couponID, req.PassengerID, xid)
	return true, nil
}

// Commit uses the coupon: frozen -> used. Idempotent via status check.
func (s *CouponService) Commit(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	couponID, ok := couponRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Coupon-Confirm] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	_, err := common.DB.Exec("UPDATE coupons SET status=2 WHERE id=? AND status=1", couponID)
	if err != nil {
		return false, fmt.Errorf("coupon confirm failed: %v", err)
	}
	couponRecords.Delete(bac.Xid)
	log.Infof("[Coupon-Confirm] coupon %d used, xid=%s", couponID, bac.Xid)
	return true, nil
}

// Rollback unfreezes the coupon: frozen -> available. Idempotent via status check.
func (s *CouponService) Rollback(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	couponID, ok := couponRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Coupon-Cancel] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	_, err := common.DB.Exec("UPDATE coupons SET status=0 WHERE id=? AND status=1", couponID)
	if err != nil {
		return false, fmt.Errorf("coupon cancel failed: %v", err)
	}
	couponRecords.Delete(bac.Xid)
	log.Infof("[Coupon-Cancel] coupon %d unfrozen, xid=%s", couponID, bac.Xid)
	return true, nil
}
