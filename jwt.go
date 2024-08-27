package common

import (
	"crypto/rsa"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func AuthCheckCtx(ctx echo.Context, publicKey *rsa.PublicKey, partnerID string, userID string) (string, *Error) {
	if ctx.Get("bearerAuth.Scopes") == nil {
		return "", nil // if no auth is required we return true
	}
	scopes := ctx.Get("bearerAuth.Scopes").([]string)
	token, err := authTokenFromHeader(ctx)
	if err != nil {
		return "", NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("failed to get auth token from header")
	}
	return token, authCheck(publicKey, token, partnerID, userID, scopes)
}

func GetRole(strToken string, publicKey *rsa.PublicKey) (string, *Error) {
	jwtToken, err := parseToken(publicKey, strToken)
	if err != nil {
		return "", err
	}
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", NewError(http.StatusInternalServerError).Msg("failed to get claims from token")
	}
	role := claims["role"].(string)

	// Translate from old roles to new ones
	if role == "manager" {
		role = "coach"
	} else if role == "hr" {
		role = "gm"
	}
	return role, nil
}

func GetPartnerAndUserFromToken(tokenStr string, publicKey *rsa.PublicKey) (partnerId string, userId string, role string, err error) {
	jwtToken, err := parseToken(publicKey, tokenStr)
	if err != nil {
		return "", "", "", NewError(http.StatusUnauthorized).Msg("invalid token")
	}

	// Get the Claims
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", "", NewError(http.StatusUnauthorized).Msg("failed to parse Claims")
	}

	if _, ok := claims["partnerId"]; !ok {
		return "", "", "", NewError(http.StatusUnauthorized).Msg("missing partnerId claim")
	}
	if _, ok := claims["userId"]; !ok {
		return "", "", "", NewError(http.StatusUnauthorized).Msg("missing userId claim")
	}
	if _, ok := claims["role"]; !ok {
		return "", "", "", NewError(http.StatusUnauthorized).Msg("missing role claim")
	}

	return claims["partnerId"].(string), claims["userId"].(string), claims["role"].(string), nil
}

func authCheck(publicKey *rsa.PublicKey, tokenStr string, partnerID string, userID string, scopes []string) *Error {
	jwtToken, err := parseToken(publicKey, tokenStr)
	if err != nil {
		return err.Str("partnerID", partnerID).Str("userID", userID).Msg("invalid token")
	}
	if !jwtToken.Valid {
		return NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("invalid token")
	}

	// Get the Claims
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("failed to parse Claims")
	}

	if claims["apiRoles"] != nil {
		if len(scopes) > 0 && scopes[0] != "" {
			apiRoles := make(map[string]bool)
			roles := claims["apiRoles"].([]interface{})
			for i := range roles {
				apiRoles[roles[i].(string)] = true
			}

			roleMatch := false
			for _, scope := range scopes {
				if apiRoles[scope] {
					roleMatch = true
				}
			}
			if !roleMatch {
				return NewError(http.StatusUnauthorized).Msg("apiUser has no claim for any of the defined scopes")
			}
		}
		return nil
	}

	// Check that the PartnerID in the URL-path matches the one in the JwtToken
	if partnerID != "" && claims["partnerId"] != partnerID {
		return NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("partnerId mismatch")
	}

	// Check that the UserID in the URL-path matches the one in the JwtToken
	if userID != "" && claims["userId"] != userID {
		return NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("userId mismatch")
	}

	// Check that user has one of the roles defined in security scope (if it's not empty)
	if len(scopes) > 0 && scopes[0] != "" {
		role := claims["role"]

		// Translate from old roles to new ones
		if role == "manager" {
			role = "coach"
		} else if role == "hr" {
			role = "gm"
		}

		if role == nil {
			return NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("user has no role defined in token")
		}
		roleMatch := false
		for _, scope := range scopes {
			if scope == role {
				roleMatch = true
			}
		}
		if !roleMatch {
			return NewError(http.StatusUnauthorized).Str("partnerID", partnerID).Str("userID", userID).Msg("user has no claim for any of the defined scopes")
		}
	}
	return nil
}

func parseToken(verifyKey *rsa.PublicKey, tokenString string) (*jwt.Token, *Error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return verifyKey, nil
	})
	if err != nil {
		return nil, NewErrorE(http.StatusUnauthorized, err).Msg("invalid token. Failed parsing")
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
