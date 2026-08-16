package jwt

import (
	"testing"
	"time"

	"we-chat/internal/config"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

func TestParseTokenRejectsUnexpectedHMACAlgorithm(t *testing.T) {
	config.AppConfig = &config.Config{JWT: config.JWTConfig{Secret: "diagnosis-secret", ExpireHours: 1}}
	claims := Claims{UserID: "u-42", RegisteredClaims: jwtv5.RegisteredClaims{ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour))}}
	forged := jwtv5.NewWithClaims(jwtv5.SigningMethodHS384, claims)
	raw, err := forged.SignedString([]byte(config.AppConfig.JWT.Secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(raw); err == nil {
		t.Fatal("HS384 token was accepted although this service is configured for HS256")
	}
}
