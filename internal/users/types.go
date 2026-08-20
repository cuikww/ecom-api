package users

import "time"

type CreateUserParam struct {
	FullName string `json:"fullName" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}
type LoginUserParam struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	User        User   `json:"user"`
	AccessToken string `json:"accessToken"`
}

type UpdateProfileRequest struct {
	FullName string `json:"fullName" binding:"required,min=3,max=100"`
	Phone    string `json:"phone" binding:"required,min=10,max=15"`
	Address  string `json:"address" binding:"required,min=10"`
}

type UserProfileResponse struct {
	ID        int64     `json:"id"`
	FullName  string    `json:"fullName"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"createdAt"`
}
