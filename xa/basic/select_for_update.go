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

	"seata.apache.org/seata-go/pkg/tm"
)

func selectForUpdateData(ctx context.Context) error {
	sql := "select id, user_id from order_tbl where id=? for update"
	ret, err := db.ExecContext(ctx, sql, 333)
	if err != nil {
		fmt.Printf("select for udpate failed, err:%v\n", err)
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("select for udpate failed, err:%v\n", err)
		return err
	}
	fmt.Printf("select for udpate success： %d.\n", rows)
	return nil
}

func sampleSelectForUpdate(ctx context.Context) {
	tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx_SelectForUpdate",
		Timeout: time.Second * 30,
	}, selectForUpdateData)
}
