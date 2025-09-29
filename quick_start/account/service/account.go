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
	"errors"
	"fmt"

	"gorm.io/gorm"
	"seata.apache.org/seata-go-samples/quick_start/account/model"
)

type AccountService struct {
	db *gorm.DB
}

func NewAccountService(db *gorm.DB) *AccountService {
	return &AccountService{db: db}
}

func (a *AccountService) Deduct(ctx context.Context, account model.Account) error {
	var userAccount model.Account
	err := a.db.WithContext(ctx).Where("user_id = ?", account.UserID).First(&userAccount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("account not found for user_id: %d", account.UserID)
		}
		return fmt.Errorf("failed to query account: %w", err)
	}

	if userAccount.Balance < account.Balance {
		return fmt.Errorf("insufficient balance: current balance %d, required %d", userAccount.Balance, account.Balance)
	}
	// 计算扣减后的新余额
	newBalance := userAccount.Balance - account.Balance
	err = a.db.WithContext(ctx).Model(&model.Account{}).
		Where("user_id = ?", account.UserID).
		Update("balance", newBalance).Error
	if err != nil {
		return fmt.Errorf("failed to deduct balance: %w", err)
	}
	return nil
}
