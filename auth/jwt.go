package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
)

type JwtTokenVerifier interface {
	VerifyToken(token string) (map[string]interface{}, error)
}

type JwtTokenVerify struct {
	verifyKey *rsa.PublicKey
}

var _ JwtTokenVerifier = &JwtTokenVerify{}

func NewJwtTokenVerify(verifyKey *rsa.PublicKey) JwtTokenVerifier {
	return &JwtTokenVerify{verifyKey: verifyKey}
}

// Parse, validate  a token.
// keyFunc will receive the token string and return claims data as map[string]interface{}.
// If everything is okay, err will be nil
// Example:
// verifier := NewJwtTokenVerify(publicKey)
// data, err := auth.VerifyToken("tokenString")
// if err != nil {
// 		fmt.Errorf("failed with err: %v",err)
// }
// partnerID := data["partnerId"]
func (j *JwtTokenVerify) VerifyToken(tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.verifyKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if err := claims.Valid(); err != nil {
			return nil, err
		}
		return claims, nil
	}
	return nil, errors.New("token claims invalid")
}
