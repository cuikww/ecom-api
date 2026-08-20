package users

import "time"

const (
	RoleAdmin    = "admin"
	RoleCustomer = "customer"
)

type User struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	FullName  string    `gorm:"column:fullname;not null" json:"fullname"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Role      string    `gorm:"column:role;not null;default:'customer'" json:"role"`
	Phone     string    `gorm:"column:phone" json:"phone"`
	Address   string    `gorm:"column:address" json:"address"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (User) TableName() string {
	return "users"
}
