package users

import "time"

type User struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	FullName  string    `gorm:"column:fullname;not null" json:"fullname"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Phone     string    `gorm:"column:phone" json:"phone"`
	Address   string    `gorm:"column:address" json:"address"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (User) TableName() string {
	return "users"
}
