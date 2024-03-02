// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/seata-go/pkg/tm"
)

func insertOnUpdateData(ctx context.Context) error {
	sql := "insert into order_tbl (id, user_id, commodity_code, count, money, descs) values (?, ?, ?, ?, ?, ?) " +
		"on duplicate key update descs=?"
	ret, err := db.ExecContext(ctx, sql, 1, "NO-100001", "C100000", 100, nil, "init desc", fmt.Sprintf("insert on update descs %d", time.Now().Unix()))
	if err != nil {
		fmt.Printf("insert on update failed, err:%v\n", err)
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("insert on update failed, err:%v\n", err)
		return err
	}
	fmt.Printf("insert on update success： %d.\n", rows)
	return nil
}

func sampleInsertOnUpdate(ctx context.Context) {
	tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx_InsertOnUpdate",
		Timeout: time.Second * 30,
	}, insertOnUpdateData)
}
