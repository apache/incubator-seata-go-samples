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

type CapacityRequest struct {
	CapacityID int64 `json:"capacity_id"`
}

type CapacityService struct{}

// capacityRecords tracks xid -> capacityID for Commit/Rollback lookup.
var capacityRecords sync.Map

func (s *CapacityService) GetActionName() string {
	return "CapacityService"
}

// Prepare reserves a vehicle slot: reserved_slots + 1 (Try phase).
func (s *CapacityService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	req, ok := params.(CapacityRequest)
	if !ok {
		return false, fmt.Errorf("invalid params type %T, want CapacityRequest", params)
	}
	result, err := common.DB.ExecContext(ctx,
		"UPDATE vehicle_capacity SET reserved_slots=reserved_slots+1 WHERE id=? AND reserved_slots<total_slots",
		req.CapacityID)
	if err != nil {
		return false, fmt.Errorf("capacity prepare failed: %v", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return false, fmt.Errorf("vehicle capacity %d full, no slots available", req.CapacityID)
	}
	xid := tm.GetXID(ctx)
	capacityRecords.Store(xid, req.CapacityID)
	log.Infof("[Capacity-Try] reserved 1 slot on vehicle capacity %d, xid=%s", req.CapacityID, xid)
	return true, nil
}

// Commit keeps the reservation as-is (reservation is the final state).
func (s *CapacityService) Commit(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	capacityID, ok := capacityRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Capacity-Confirm] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	capacityRecords.Delete(bac.Xid)
	log.Infof("[Capacity-Confirm] slot reservation confirmed on capacity %d, xid=%s", capacityID, bac.Xid)
	return true, nil
}

// Rollback releases the vehicle slot: reserved_slots - 1. Idempotent via reserved_slots>0 check.
func (s *CapacityService) Rollback(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	capacityID, ok := capacityRecords.Load(bac.Xid)
	if !ok {
		log.Infof("[Capacity-Cancel] no record for xid=%s, idempotent skip", bac.Xid)
		return true, nil
	}
	_, err := common.DB.Exec(
		"UPDATE vehicle_capacity SET reserved_slots=reserved_slots-1 WHERE id=? AND reserved_slots>0",
		capacityID)
	if err != nil {
		return false, fmt.Errorf("capacity cancel failed: %v", err)
	}
	capacityRecords.Delete(bac.Xid)
	log.Infof("[Capacity-Cancel] slot released on capacity %d, xid=%s", capacityID, bac.Xid)
	return true, nil
}
