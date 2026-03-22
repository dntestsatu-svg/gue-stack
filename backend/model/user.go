package model

import "time"

type UserRole string

const (
	UserRoleDev        UserRole = "dev"
	UserRoleSuperAdmin UserRole = "superadmin"
	UserRoleAdmin      UserRole = "admin"
	UserRoleUser       UserRole = "user"
)

type User struct {
	ID           uint64    `json:"id" gorm:"column:id;primaryKey;autoIncrement;type:bigint unsigned"`
	Name         string    `json:"name" gorm:"column:name;type:varchar(100);not null"`
	Email        string    `json:"email" gorm:"column:email;type:varchar(255);not null;uniqueIndex:uniq_users_email"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;type:varchar(255);not null"`
	Role         UserRole  `json:"role" gorm:"column:role;type:enum('dev','superadmin','admin','user');not null;default:user;index:idx_users_role"`
	IsActive     bool      `json:"is_active" gorm:"column:is_active;type:tinyint(1);not null;default:1;index:idx_users_is_active"`
	CreatedBy    *uint64   `json:"created_by,omitempty" gorm:"column:created_by;type:bigint unsigned;index:idx_users_created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Tokos []Toko `json:"tokos,omitempty" gorm:"many2many:toko_users;joinForeignKey:UserID;joinReferences:TokoID"`
	Banks []Bank `json:"banks,omitempty" gorm:"foreignKey:UserID;references:ID"`
}

func (User) TableName() string {
	return "users"
}
