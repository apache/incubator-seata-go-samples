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
)

func insertOnUpdateDataFail(ctx context.Context) error {
	// generate an err : insert on update sql requires primary key insert column
	sql := "insert into order_tbl (user_id, commodity_code, count, money, descs) values (?, ?, ?, ?, ?) " +
		"on duplicate key update descs=?"
	ret, err := db.ExecContext(ctx, sql, "NO-100001", "C100000", 100, nil, "init desc", fmt.Sprintf("insert on update descs %d", time.Now().Unix()))
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return nil
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return nil
	}
	fmt.Printf("update success： %d.\n", rows)
	return nil
}
