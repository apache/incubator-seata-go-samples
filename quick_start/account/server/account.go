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

package server

import (
	"context"
	"fmt"
	"strconv"

	"seata.apache.org/seata-go-samples/quick_start/account/model"
	"seata.apache.org/seata-go-samples/quick_start/account/service"
	pb "seata.apache.org/seata-go-samples/quick_start/api"
)

type AccountServer struct {
	svc *service.AccountService
	pb.UnimplementedAccountServiceServer
}

func NewAccountServer(svc *service.AccountService) *AccountServer {
	return &AccountServer{
		svc: svc,
	}
}

func (a *AccountServer) Deduct(ctx context.Context, request *pb.AccountDeductRequest) (*pb.AccountResponse, error) {
	userID, err := strconv.ParseInt(request.UserId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	account := model.Account{
		UserID:  userID,
		Balance: request.Money,
	}

	if err := a.svc.Deduct(ctx, account); err != nil {
		return nil, err
	}

	return &pb.AccountResponse{
		UserId:       request.UserId,
		Balance:      0,
		FreezeAmount: 0,
	}, nil
}
