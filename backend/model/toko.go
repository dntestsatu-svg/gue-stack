package model

import "time"

type Toko struct {
	ID          uint64  `json:"id" gorm:"column:id;primaryKey;autoIncrement;type:bigint unsigned"`
	Name        string  `json:"name" gorm:"column:name;type:varchar(255);not null"`
	Token       string  `json:"token" gorm:"column:token;type:varchar(255);not null;uniqueIndex:uniq_tokos_token"`
	Charge      int     `json:"charge" gorm:"column:charge;type:int;not null;default:2"`
	CallbackURL *string `json:"callback_url,omitempty" gorm:"column:callback_url;type:varchar(255)"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Balance      *Balance      `json:"balance,omitempty" gorm:"foreignKey:TokoID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Transactions []Transaction `json:"transactions,omitempty" gorm:"foreignKey:TokoID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Users        []User        `json:"users,omitempty" gorm:"many2many:toko_users;joinForeignKey:TokoID;joinReferences:UserID"`
}

func (Toko) TableName() string {
	return "tokos"
}
