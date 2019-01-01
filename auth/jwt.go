package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
)

// Parse, validate  a token.
// keyFunc will receive the token string and return claims data as map[string]interface{}.
// If everything is okay, err will be nil
// Example:
// data, err := auth.VerifyToken(key,"tokenString")
// if err != nil {
// 		fmt.Errorf("failed with err: %v",err)
// }
// partnerID := data["partnerId"]
//
func VerifyToken(verifyKey *rsa.PublicKey, tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return verifyKey, nil
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
