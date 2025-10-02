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
	"sync"

	"seata.apache.org/seata-go/pkg/rm/tcc"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	tccService     *tcc.TCCServiceProxy
	tccServiceOnce sync.Once
)

type OrderService struct{}

func NewOrderServiceProxy() *tcc.TCCServiceProxy {
	if tccService != nil {
		return tccService
	}
	tccServiceOnce.Do(func() {
		var err error
		tccService, err = tcc.NewTCCServiceProxy(&OrderService{})
		if err != nil {
			panic(fmt.Errorf("get OrderService tcc service proxy error, %v", err.Error()))
		}
	})
	return tccService
}

func (o OrderService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	log.Infof("OrderService Prepare, param %v", params)
	return true, nil
}

func (o OrderService) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("OrderService Commit, param %v", businessActionContext)
	return true, nil
}

func (o OrderService) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("OrderService Rollback, param %v", businessActionContext)
	return true, nil
}

func (o OrderService) GetActionName() string {
	return "OrderService"
}
