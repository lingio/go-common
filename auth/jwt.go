package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
)

type TokenVerifier interface {
	VerifyToken(token string) (map[string]interface{}, error)
	GetPrincipalWithToken(tokenString string, principal AuthenticatePrincipal) (AuthenticatePrincipal, error)
}

type AuthenticatePrincipal interface {
	MapData(map[string]interface{}) error
}

type RsaTokenVerifier struct {
	verifyKey *rsa.PublicKey
}

var _ TokenVerifier = &RsaTokenVerifier{}

// Create new struct TokenVerifier
// keyFunc will receive the public key and return struct as TokenVerifier.
// Example:
// key, err := ioutil.ReadFile("lingio-rsa256")
// if err != nil {
// 		os.Exit(1)
// }
// publicKey, err := jwt.ParseRSAPublicKeyFromPEM(pkey)
// if err != nil {
// 		os.Exit(1)
// }
// verifier := NewTokenVerify(publicKey)
func NewTokenVerify(verifyKey *rsa.PublicKey) TokenVerifier {
	return &RsaTokenVerifier{verifyKey: verifyKey}
}

// Parse, validate  a token.
// keyFunc will receive the token string and return claims data as map[string]interface{}.
// If everything is okay, err will be nil
// Example:
// verifier := NewTokenVerify(publicKey)
// data, err := verifier.VerifyToken("tokenString")
// if err != nil {
// 		fmt.Errorf("failed with err: %v",err)
// }
// partnerID := data["partnerId"]
func (j *RsaTokenVerifier) VerifyToken(tokenString string) (map[string]interface{}, error) {
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

// Parse, validate  a token, then use claims for map it into principal struct.
// keyFunc will receive the token string, principal interface and return mapped principal data as AuthenticatePrincipal.
// If everything is okay, err will be nil
// Example:
// verifier := NewTokenVerify(publicKey)
// principal := models.AuthenticatePrincipal{}
// mappedPrincipal, err := verifier.GetPrincipalWithToken("tokenString",principal)
// if err != nil {
// 		fmt.Errorf("failed with err: %v",err)
// }
// partnerID := mappedPrincipal.PartnerID
func (j *RsaTokenVerifier) GetPrincipalWithToken(tokenString string, principal AuthenticatePrincipal) (AuthenticatePrincipal, error) {
	claims, err := j.VerifyToken(tokenString)
	if err != nil {
		return nil, err
	}
	if err := principal.MapData(claims); err != nil {
		return nil, err
	}
	return principal, nil
}
