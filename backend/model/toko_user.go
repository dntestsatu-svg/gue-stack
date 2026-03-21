package model

import "time"

type TokoUser struct {
	ID        uint64 `json:"id" gorm:"column:id;primaryKey;autoIncrement;type:bigint unsigned"`
	UserID    uint64 `json:"user_id" gorm:"column:user_id;type:bigint unsigned;not null;index:idx_toko_users_user_id;uniqueIndex:uniq_toko_users_user_toko"`
	TokoID    uint64 `json:"toko_id" gorm:"column:toko_id;type:bigint unsigned;not null;index:idx_toko_users_toko_id;uniqueIndex:uniq_toko_users_user_toko"`
	CreatedAt time.Time
	UpdatedAt time.Time

	User User `json:"user" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Toko Toko `json:"toko" gorm:"foreignKey:TokoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (TokoUser) TableName() string {
	return "toko_users"
}
