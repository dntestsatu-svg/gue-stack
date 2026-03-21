package model

import "time"

type Payment struct {
	ID            uint64  `json:"id" gorm:"column:id;primaryKey;autoIncrement;type:bigint unsigned"`
	BankCode      string  `json:"bank_code" gorm:"column:bank_code;type:varchar(10);not null;index:idx_payments_bank_code"`
	BankName      string  `json:"bank_name" gorm:"column:bank_name;type:varchar(255);not null;index:idx_payments_bank_name"`
	BankSwiftCode *string `json:"bank_swift_code,omitempty" gorm:"column:bank_swift_code;type:varchar(255)"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Payment) TableName() string {
	return "payments"
}
