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
	CouponID int64 `json:"coupon_id"`
}

type CouponService struct{}

// couponRecords tracks xid -> couponID for Commit/Rollback lookup.
var couponRecords sync.Map

func (s *CouponService) GetActionName() string {
	return "CouponService"
}

// Prepare freezes the coupon: available -> frozen (Try phase).
func (s *CouponService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	req := params.(CouponRequest)
	result, err := common.DB.ExecContext(ctx,
		"UPDATE coupons SET status=1 WHERE id=? AND status=0", req.CouponID)
	if err != nil {
		return false, fmt.Errorf("coupon prepare failed: %v", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return false, fmt.Errorf("coupon %d not available", req.CouponID)
	}
	xid := tm.GetXID(ctx)
	couponRecords.Store(xid, req.CouponID)
	log.Infof("[Coupon-Try] froze coupon %d, xid=%s", req.CouponID, xid)
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
