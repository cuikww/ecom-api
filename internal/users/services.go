package users

import (
	"context"
	"errors"

	"ecom-api/internal/auth"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("email atau password salah")

type Service interface {
	CreateUser(ctx context.Context, p CreateUserParam) (User, error)
	LoginUser(ctx context.Context, p LoginUserParam) (LoginResponse, error)
}

type service struct {
	db         *gorm.DB
	jwtService *auth.JWTService
}

func NewService(db *gorm.DB, jwtService *auth.JWTService) Service {
	return &service{db: db, jwtService: jwtService}
}

func (s *service) CreateUser(ctx context.Context, p CreateUserParam) (User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	user := User{FullName: p.FullName, Email: p.Email, Password: string(hashed)}
	err = s.db.WithContext(ctx).Create(&user).Error
	return user, err
}

func (s *service) LoginUser(ctx context.Context, p LoginUserParam) (LoginResponse, error) {
	var user User
	err := s.db.WithContext(ctx).Where("email = ?", p.Email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return LoginResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResponse{}, err // error DB asli tetap naik, tidak ditelan
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(p.Password)); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	accessToken, err := s.jwtService.GenerateToken(int64(user.ID))
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{User: user, AccessToken: accessToken}, nil
}
