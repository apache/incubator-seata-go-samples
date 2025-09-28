package model

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
