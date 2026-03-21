package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Balance struct {
	ID        uint64          `json:"id" gorm:"column:id;primaryKey;autoIncrement;type:bigint unsigned"`
	TokoID    uint64          `json:"toko_id" gorm:"column:toko_id;type:bigint unsigned;not null;uniqueIndex:uniq_balances_toko_id"`
	Pending   decimal.Decimal `json:"pending" gorm:"column:pending;type:decimal(15,2);not null;default:0.00"`
	Available decimal.Decimal `json:"available" gorm:"column:available;type:decimal(15,2);not null;default:0.00"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Toko Toko `json:"toko" gorm:"foreignKey:TokoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (Balance) TableName() string {
	return "balances"
}
