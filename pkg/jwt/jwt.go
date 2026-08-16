package jwt

import (
	"errors"
	"fmt"
	"time"

	"we-chat/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, username, email string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.AppConfig.JWT.ExpireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	method, err := configuredSigningMethod()
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(method, claims)
	return token.SignedString([]byte(config.AppConfig.JWT.Secret))
}

func ParseToken(tokenString string) (*Claims, error) {
	expected, err := configuredSigningMethod()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != expected.Alg() {
			return nil, fmt.Errorf("unexpected signing algorithm %q", token.Method.Alg())
		}
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unsupported signing method %T", token.Method)
		}
		return []byte(config.AppConfig.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func configuredSigningMethod() (jwt.SigningMethod, error) {
	algorithm := config.AppConfig.JWT.Algorithm
	if algorithm == "" {
		algorithm = jwt.SigningMethodHS256.Alg()
	}
	method := jwt.GetSigningMethod(algorithm)
	if method == nil {
		return nil, fmt.Errorf("unsupported JWT algorithm %q", algorithm)
	}
	if _, ok := method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("JWT algorithm %q is not HMAC", algorithm)
	}
	return method, nil
}
