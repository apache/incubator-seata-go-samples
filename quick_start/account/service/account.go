package service

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"seata.apache.org/seata-go-samples/quick_start/account/model"
)

type AccountService struct {
	db *gorm.DB
}

func NewAccountService(db *gorm.DB) *AccountService {
	return &AccountService{db: db}
}

func (a *AccountService) Deduct(ctx context.Context, account model.Account) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var userAccount model.Account
		err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", account.UserID).First(&userAccount).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("account not found for user_id: %d", account.UserID)
			}
			return fmt.Errorf("failed to query account: %w", err)
		}

		if userAccount.Balance < account.Balance {
			return fmt.Errorf("insufficient balance: current balance %d, required %d", userAccount.Balance, account.Balance)
		}

		newBalance := userAccount.Balance - account.Balance
		err = tx.WithContext(ctx).Model(&model.Account{}).
			Where("user_id = ?", account.UserID).
			Update("balance", newBalance).Error
		if err != nil {
			return fmt.Errorf("failed to deduct balance: %w", err)
		}

		return nil
	})
}
