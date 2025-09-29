package model

import "gorm.io/gorm"

type Account struct {
	ID      int64 `gorm:"column:id;autoIncrement;primaryKey"`
	UserID  int64 `gorm:"column:user_id"`
	Balance int64 `gorm:"column:balance"`
	Freeze  int64 `gorm:"column:freeze"`
	Ctime   int64 `gorm:"column:ctime"`
	Utime   int64 `gorm:"column:utime"`
}

func (Account) TableName() string {
	return "accounts"
}

func InitTable(db *gorm.DB) {
	if err := db.AutoMigrate(&Account{}); err != nil {
		panic("auto migrate error")
	}
}
