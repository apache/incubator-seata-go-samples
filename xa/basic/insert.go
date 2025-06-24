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
	"time"

	"seata.apache.org/seata-go/pkg/tm"
)

func insertData(ctx context.Context) error {
	sql := "INSERT INTO `order_tbl` ( `user_id`, `commodity_code`, `count`, `money`, `descs`) VALUES (?, ?, ?, ?, ?);"
	ret, err := db.ExecContext(ctx, sql, "NO-100001", "C100000", 100, nil, "init desc")
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		return err
	}
	fmt.Printf("insert success： %d.\n", rows)
	return nil
}

func sampleInsert(ctx context.Context) {
	_ = tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "XASampleLocalGlobalTx_Insert",
		Timeout: time.Second * 30,
	}, insertData)
}
