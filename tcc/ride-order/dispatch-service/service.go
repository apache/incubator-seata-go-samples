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

type DispatchRequest struct {
	OrderID      int64 `json:"order_id"`
	SimulateFail bool  `json:"simulate_fail"`
}

type DispatchService struct{}

// dispatchRecords tracks xid -> driverID for Commit/Rollback lookup.
var dispatchRecords sync.Map

func (s *DispatchService) GetActionName() string {
	return "DispatchService"
}

// Prepare reserves an available driver (Try phase).
// When SimulateFail is true, returns an error to trigger global rollback.
func (s *DispatchService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	req := params.(DispatchRequest)
	if req.SimulateFail {
		log.Infof("[Dispatch-Try] SIMULATED FAILURE: no available driver, xid=%s", tm.GetXID(ctx))
		return false, fmt.Errorf("no available driver found (simulated failure)")
	}

	var driverID int64
	err := common.DB.QueryRowContext(ctx,
		"SELECT id FROM drivers WHERE status=0 LIMIT 1").Scan(&driverID)
	if err != nil {
		return false, fmt.Errorf("no available driver: %v", err)
	}
	_, err = common.DB.ExecContext(ctx,
		"UPDATE drivers SET status=1, reserved_order_id=? WHERE id=? AND status=0",
		req.OrderID, driverID)
	if err != nil {
		return false, fmt.Errorf("dispatch prepare failed: %v", err)
	}
	xid := tm.GetXID(ctx)
	dispatchRecords.Store(xid, driverID)
	log.Infof("[Dispatch-Try] reserved driver %d for order %d, xid=%s", driverID, req.OrderID, xid)
	return true, nil
}

// Commit confirms the driver: reserved -> busy. Idempotent via status check.
func (s *DispatchService) Commit(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	driverID, ok := dispatchRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Dispatch-Confirm] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	_, err := common.DB.Exec("UPDATE drivers SET status=2 WHERE id=? AND status=1", driverID)
	if err != nil {
		return false, fmt.Errorf("dispatch confirm failed: %v", err)
	}
	dispatchRecords.Delete(bac.Xid)
	log.Infof("[Dispatch-Confirm] driver %d now busy, xid=%s", driverID, bac.Xid)
	return true, nil
}

// Rollback releases the driver: reserved -> available. Idempotent via status check.
func (s *DispatchService) Rollback(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	driverID, ok := dispatchRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Dispatch-Cancel] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	_, err := common.DB.Exec("UPDATE drivers SET status=0, reserved_order_id=NULL WHERE id=? AND status=1", driverID)
	if err != nil {
		return false, fmt.Errorf("dispatch cancel failed: %v", err)
	}
	dispatchRecords.Delete(bac.Xid)
	log.Infof("[Dispatch-Cancel] driver %d released, xid=%s", driverID, bac.Xid)
	return true, nil
}
