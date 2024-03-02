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
	"database/sql"
	"fmt"
	"time"

	"github.com/apache/seata-go-samples/util"
	"github.com/apache/seata-go/pkg/client"
)

var db *sql.DB

func main() {
	client.InitPath("../../conf/seatago.yml")
	db = util.GetXAMySqlDb()
	ctx := context.Background()

	// sample: insert
	sampleInsert(ctx)

	// sample: insert on update
	//sampleInsertOnUpdate(ctx)

	// sample: select for udpate
	//sampleSelectForUpdate(ctx)

	<-make(chan struct{})
}

func deleteData(ctx context.Context) error {
	sql := "delete from order_tbl where id=?"
	ret, err := db.ExecContext(ctx, sql, 2)
	if err != nil {
		fmt.Printf("delete failed, err:%v\n", err)
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("delete failed, err:%v\n", err)
		return err
	}
	fmt.Printf("delete success： %d.\n", rows)
	return nil
}

func updateData(ctx context.Context) error {
	sql := "update order_tbl set descs=? where id=?"
	ret, err := db.ExecContext(ctx, sql, fmt.Sprintf("NewDescs-%d", time.Now().UnixMilli()), 1)
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return err
	}
	fmt.Printf("update success： %d.\n", rows)
	return nil
}
