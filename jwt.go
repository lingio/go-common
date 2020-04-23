package common

import (
	"crypto/rsa"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"net/http"
)

func AuthCheckCtx(ctx echo.Context, publicKey *rsa.PublicKey, partnerID string, userID string) (bool, *Error) {
	if ctx.Get("bearerAuth.Scopes") == nil {
		return true, nil // if no auth is required we return true
	}
	scopes := ctx.Get("bearerAuth.Scopes").([]string)
	token, err := authTokenFromHeader(ctx)
	if err != nil {
		return false, NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("failed to get auth token from header")
	}
	return authCheck(publicKey, token, partnerID, userID, scopes)
}

func authCheck(publicKey *rsa.PublicKey, tokenStr string, partnerID string, userID string, scopes []string) (bool, *Error) {
	jwtToken, err := parseToken(publicKey, tokenStr)
	if err != nil {
		return false, NewErrorE(http.StatusUnauthorized, err).Str("partnerID", partnerID).Str("userID", userID).Msg("invalid token. Failed parsing")
	}
	if !jwtToken.Valid {
		return false, NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("invalid token")
	}

	// Get the Claims
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return false, NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("failed to parse Claims")
	}

	// Check that the PartnerID in the URL-path matches the one in the JwtToken
	if partnerID != "" && claims["partnerId"] != partnerID {
		return false, NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("partnerId mismatch")
	}

	// Check that the UserID in the URL-path matches the one in the JwtToken
	if userID != "" && claims["userId"] != userID {
		return false, NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("userId mismatch")
	}

	// Check that user has one of the roles defined in security scope (if it's not empty)
	if len(scopes) > 0 && scopes[0] != "" {
		role := claims["role"]
		if role == nil {
			return false, NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("user has no role defined in token")
		}
		roleMatch := false
		for _, scope := range scopes {
			if scope == role {
				roleMatch = true
			}
		}
		if !roleMatch {
			return false, NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("user has no claim for any of the defined scopes")
		}
	}

	return true, nil
}

func parseToken(verifyKey *rsa.PublicKey, tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return verifyKey, nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func authTokenFromHeader(c echo.Context) (string, *Error) {
	auth := c.Request().Header.Get("Authorization")
	authScheme := "Bearer"
	l := len(authScheme)
	if len(auth) > l+1 && auth[:l] == authScheme {
		return auth[l+1:], nil
	}
	return "", NewError(http.StatusBadRequest).Msg("authorization header missing")
}
