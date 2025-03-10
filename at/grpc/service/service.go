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
	"database/sql"
	"fmt"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"seata.apache.org/seata-go-samples/at/grpc/pb"
	"time"

	sql2 "seata.apache.org/seata-go/pkg/datasource/sql"
)

var (
	db *sql.DB
)

func InitService() {
	var err error
	db, err = sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true&interpolateParams=true")
	if err != nil {
		panic("init service error")
	}
}

type GrpcBusinessService struct {
	__.UnimplementedATServiceBusinessServer
}

func (service GrpcBusinessService) UpdateDataSuccess(ctx context.Context, params *__.Params) (*wrapperspb.BoolValue, error) {
	sql := "update order_tbl set descs=? where id=?"
	ret, err := db.ExecContext(ctx, sql, fmt.Sprintf("NewDescs1-%d", time.Now().UnixMilli()), 1)
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return wrapperspb.Bool(false), err
	}

	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return wrapperspb.Bool(false), err
	}
	fmt.Printf("update success： %d.\n", rows)
	return wrapperspb.Bool(true), nil
}
