package model

import "time"

type Bank struct {
	ID            uint64    `json:"id" gorm:"column:id;primaryKey;autoIncrement;type:bigint unsigned"`
	UserID        uint64    `json:"user_id" gorm:"column:user_id;type:bigint unsigned;not null;index:idx_banks_user_id"`
	PaymentID     uint64    `json:"payment_id" gorm:"column:payment_id;type:bigint unsigned;not null"`
	BankCode      string    `json:"bank_code" gorm:"column:bank_code;type:varchar(10);not null"`
	BankName      string    `json:"bank_name" gorm:"column:bank_name;type:varchar(255);not null;index:idx_banks_bank_name"`
	AccountName   string    `json:"account_name" gorm:"column:account_name;type:varchar(255);not null;index:idx_banks_account_name"`
	AccountNumber string    `json:"account_number" gorm:"column:account_number;type:varchar(64);not null;index:idx_banks_account_number"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	User    User    `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
	Payment Payment `json:"payment,omitempty" gorm:"foreignKey:PaymentID;references:ID"`
}

func (Bank) TableName() string {
	return "banks"
}
