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
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
	pb "seata.apache.org/seata-go-samples/quick_start/api"
	"seata.apache.org/seata-go-samples/quick_start/order/model"
	grpc2 "seata.apache.org/seata-go/pkg/integration/grpc"
	"seata.apache.org/seata-go/pkg/tm"
)

type OrderService struct {
	db            *gorm.DB
	config        *tm.GtxConfig
	accountClient pb.AccountServiceClient
}

func NewOrderService(db *gorm.DB, config *tm.GtxConfig) *OrderService {
	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpc2.ClientTransactionInterceptor))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to account service: %v", err))
	}

	return &OrderService{
		db:            db,
		config:        config,
		accountClient: pb.NewAccountServiceClient(conn),
	}
}

func (o *OrderService) Create(ctx context.Context, order model.Order) (int64, error) {
	err := tm.WithGlobalTx(ctx, o.config, func(ctx context.Context) error {
		now := time.Now().Unix()
		order.Ctime = now
		order.Utime = now
		err := o.db.WithContext(ctx).Create(&order).Error
		if err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		_, err = o.accountClient.Deduct(ctx, &pb.AccountDeductRequest{
			UserId: order.UserID,
			Money:  order.Money,
		})
		if err != nil {
			return fmt.Errorf("failed to deduct account balance: %w", err)
		}
		return nil
	})
	return order.ID, err
}
