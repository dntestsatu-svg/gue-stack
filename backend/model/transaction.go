package model

import "time"

type TransactionType string

type TransactionStatus string

const (
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypeWithdraw TransactionType = "withdraw"

	TransactionStatusPending TransactionStatus = "pending"
	TransactionStatusSuccess TransactionStatus = "success"
	TransactionStatusFailed  TransactionStatus = "failed"
)

type Transaction struct {
	ID            uint64            `json:"id" gorm:"column:id;primaryKey;autoIncrement;type:bigint unsigned"`
	TokoID        uint64            `json:"toko_id" gorm:"column:toko_id;type:bigint unsigned;not null;index:idx_transactions_toko_id"`
	Player        *string           `json:"player,omitempty" gorm:"column:player;type:varchar(255)"`
	Code          *string           `json:"code,omitempty" gorm:"column:code;type:varchar(255)"`
	Type          TransactionType   `json:"type" gorm:"column:type;type:enum('deposit','withdraw');not null;default:deposit;index:idx_transactions_type"`
	Status        TransactionStatus `json:"status" gorm:"column:status;type:enum('pending','success','failed');not null;index:idx_transactions_status"`
	Barcode       *string           `json:"barcode,omitempty" gorm:"column:barcode;type:text"`
	Reference     *string           `json:"reference,omitempty" gorm:"column:reference;type:varchar(255);index:idx_transactions_reference"`
	Amount        uint64            `json:"amount" gorm:"column:amount;type:bigint unsigned;not null"`
	FeeWithdrawal *uint64           `json:"fee_withdrawal,omitempty" gorm:"column:fee_withdrawal;type:bigint unsigned"`
	PlatformFee   uint64            `json:"platform_fee" gorm:"column:platform_fee;type:bigint unsigned;not null;default:0"`
	Netto         uint64            `json:"netto" gorm:"column:netto;type:bigint unsigned;not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Toko Toko `json:"toko" gorm:"foreignKey:TokoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (Transaction) TableName() string {
	return "transactions"
}
