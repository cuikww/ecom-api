package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	secret     []byte
	issuer     string
	audience   string
	expiration time.Duration
}

func NewJWTService(
	secret string,
	issuer string,
	audience string,
	expiration time.Duration,
) *JWTService {
	return &JWTService{
		secret:     []byte(secret),
		issuer:     issuer,
		audience:   audience,
		expiration: expiration,
	}
}

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func (j *JWTService) GenerateToken(userID int64, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			Issuer:    j.issuer,
			Audience:  jwt.ClaimStrings{j.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)
	return token.SignedString(j.secret)
}
func (j *JWTService) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
			}

			return j.secret, nil
		},
		jwt.WithIssuer(j.issuer),
		jwt.WithAudience(j.audience),
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}
