package model

import "time"

type User struct {
	ID           uint64    `json:"id" gorm:"column:id;primaryKey;autoIncrement;type:bigint unsigned"`
	Name         string    `json:"name" gorm:"column:name;type:varchar(100);not null"`
	Email        string    `json:"email" gorm:"column:email;type:varchar(255);not null;uniqueIndex:uniq_users_email"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;type:varchar(255);not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Tokos []Toko `json:"tokos,omitempty" gorm:"many2many:toko_users;joinForeignKey:UserID;joinReferences:TokoID"`
}

func (User) TableName() string {
	return "users"
}
