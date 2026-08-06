package common

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// AuthCheckCtx performs endpoint authentication, returning an error if the endpoint is:
//   - scope-sec, but req has no token (no auth header)
//   - scope-sec, and req has token, but not valid (token not valid)
//   - scope-sec, and req has token, but no matching scopes (missing scope)
//   - token-sec, but req has no token (no auth header)
//   - token-sec, and req has token, but not valid (token not valid)
//   - open, and req has token, but not valid (token not valid)
//
// Definitions for endpoint security:
//   - `scope-sec`: has security scopes and the first is any other than "".
//   - `token-sec`: has a single security scope equal to "".
//   - `open`: has no security configuration.
//
// Note that the final error rule means that passing an invalid jwt to an open
// endpoint will result in an authentication error.
func AuthCheckCtx(ctx echo.Context, publicKey *rsa.PublicKey, partnerID string, userID string) (string, *AuthClaims, error) {
	authScopes := ctx.Get("bearerAuth.Scopes")

	// note: add other auth mechanisms here, e.g. api-key
	// using "Authorization: APIKEY lio-xx"

	token, err := authTokenFromHeader(ctx)
	if err != nil { // no auth header
		if authScopes == nil {
			return "", nil, nil // no auth required => all good
		}
		return "", nil, Errorf(err, http.StatusUnauthorized).
			Str("partnerID", partnerID).
			Str("userID", userID)
	}

	var scopes []string
	if authScopes != nil {
		scopes = authScopes.([]string)
	}

	claims, err := authCheck(publicKey, token, partnerID, userID, scopes)
	return token, claims, err
}

func authCheck(publicKey *rsa.PublicKey, tokenStr string, partnerID string, userID string, scopes []string) (*AuthClaims, error) {
	_, claims, err := ParseToken(publicKey, tokenStr)
	if err != nil {
		return nil, Errorf(err, "invalid token").Str("partnerID", partnerID).Str("userID", userID)
	}

	var (
		open         = len(scopes) == 0
		tokenSecured = len(scopes) == 1 && scopes[0] == ""
		scopeSecured = len(scopes) > 0 && scopes[0] != ""
	)

	if claims.IsServiceToken() {
		if open || tokenSecured {
			return claims, nil // tokenSecured endpoint
		}

		// scopeSecured endpoint
		apiRoles := make(map[string]struct{})
		for _, role := range claims.Roles {
			apiRoles[role] = struct{}{}
		}

		for _, scope := range scopes {
			if _, ok := apiRoles[scope]; ok {
				return claims, nil
			}
		}
		return nil, NewError(http.StatusUnauthorized).
			Str("endpointScopes", strings.Join(scopes, ", ")).
			Str("tokenScopes", strings.Join(claims.Roles, ",")).
			Msg("apiUser has no claim for any of the required scopes")
	}

	// Check that the PartnerID in the URL-path matches the one in the JwtToken
	if partnerID != "" && claims.PartnerID != partnerID {
		return nil, NewError(http.StatusUnauthorized).
			Str("partnerID", partnerID).
			Str("tokenPartnerID", claims.PartnerID).
			Msg("partnerId mismatch")
	}

	// Check that the UserID in the URL-path matches the one in the JwtToken
	if userID != "" && claims.UserID != userID {
		return nil, NewError(http.StatusUnauthorized).
			Str("userID", userID).
			Str("tokenUserID", claims.UserID).
			Msg("userId mismatch")
	}

	// Check that user has one of the roles defined in security scope (if it's not empty)
	if scopeSecured {
		if claims.Role == "" {
			return nil, NewError(http.StatusUnauthorized).
				Str("partnerID", partnerID).
				Str("userID", userID).
				Msg("user has no role defined in token")
		}
		if ok := slices.Contains(scopes, claims.Role); !ok {
			return nil, NewError(http.StatusUnauthorized).
				Str("partnerID", partnerID).
				Str("userID", userID).
				Str("role", claims.Role).
				Str("scopes", strings.Join(scopes, ",")).
				Msg("user role does not match any of the required scopes")
		}
	}
	return claims, nil
}

func ParseToken(verifyKey *rsa.PublicKey, tokenString string) (*jwt.Token, *AuthClaims, error) {
	var c = new(AuthClaims)
	token, err := jwt.ParseWithClaims(tokenString, c, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return verifyKey, nil
	})
	if err != nil {
		return token, c, NewErrorE(http.StatusUnauthorized, err).Msg("invalid token: failed parsing")
	}
	return token, c, nil
}

func authTokenFromHeader(c echo.Context) (string, error) {
	auth := c.Request().Header.Get("Authorization")
	authScheme := "Bearer"
	l := len(authScheme)
	if len(auth) > l+1 && auth[:l] == authScheme {
		return auth[l+1:], nil
	}
	return "", NewError(http.StatusBadRequest).Msg("authorization header missing")
}

func migrateRole(role string) string {
	if role == "manager" {
		role = "coach"
	} else if role == "hr" {
		role = "gm"
	}
	return role
}

// AuthClaims describes both user and service tokens.
type AuthClaims struct {
	PartnerID     string    `json:"partnerId"`          // user and service
	UserID        string    `json:"userId"`             // user and service
	DeviceID      string    `json:"deviceId,omitzero"`  // user only
	Role          string    `json:"role"`               // user and service
	ExpiresAt     time.Time `json:"exp"`                // user and service
	RefreshBefore time.Time `json:"rbf,omitzero"`       // user only
	IssuedAt      time.Time `json:"created"`            // user and service
	Roles         []string  `json:"apiRoles,omitempty"` // service only
}

func (u AuthClaims) IsServiceToken() bool {
	return u.Role == "api-user" || len(u.Roles) > 0
}

// MarshalJSON implements the json.Marshaler interface.
func (u AuthClaims) MarshalJSON() ([]byte, error) {
	raw := struct {
		PartnerID     string           `json:"partnerId"`
		UserID        string           `json:"userId"`
		DeviceID      string           `json:"deviceId,omitzero"`
		Role          string           `json:"role"`
		ExpiresAt     *jwt.NumericDate `json:"exp"`
		RefreshBefore *jwt.NumericDate `json:"rbf,omitzero"`
		IssuedAt      *jwt.NumericDate `json:"created"`
		Roles         []string         `json:"apiRoles,omitempty"`
	}{
		PartnerID: u.PartnerID,
		UserID:    u.UserID,
		DeviceID:  u.DeviceID,
		Role:      migrateRole(u.Role),
		IssuedAt:  jwt.NewNumericDate(u.IssuedAt),
		Roles:     u.Roles,
	}
	if !u.ExpiresAt.IsZero() {
		raw.ExpiresAt = jwt.NewNumericDate(u.ExpiresAt)
	}
	if !u.RefreshBefore.IsZero() {
		raw.RefreshBefore = jwt.NewNumericDate(u.RefreshBefore)
	}

	return json.Marshal(raw)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (u *AuthClaims) UnmarshalJSON(data []byte) error {
	var raw struct {
		PartnerID     string           `json:"partnerId"`
		UserID        string           `json:"userId"`
		DeviceID      string           `json:"deviceId"`
		Role          string           `json:"role"`
		ExpiresAt     *jwt.NumericDate `json:"exp"`
		RefreshBefore *jwt.NumericDate `json:"rbf"`
		IssuedAt      json.RawMessage  `json:"created"`
		Roles         []string         `json:"apiRoles"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*u = AuthClaims{
		PartnerID: raw.PartnerID,
		UserID:    raw.UserID,
		DeviceID:  raw.DeviceID,
		Role:      migrateRole(raw.Role),
		Roles:     raw.Roles,
	}
	if raw.ExpiresAt != nil {
		u.ExpiresAt = raw.ExpiresAt.Time
	}
	if raw.RefreshBefore != nil {
		u.RefreshBefore = raw.RefreshBefore.Time
	}

	if len(raw.IssuedAt) > 0 {
		if raw.IssuedAt[0] == '"' {
			// string-encoded unix (backwards-compatible)
			q := "\""
			x := bytes.TrimRight(bytes.TrimLeft(raw.IssuedAt, q), q)
			sec64, err := strconv.ParseInt(string(x), 10, 64)
			if err != nil {
				return err
			}
			u.IssuedAt = time.Unix(sec64, 0)
		} else {
			// numeric date
			var n jwt.NumericDate
			if err := json.Unmarshal(raw.IssuedAt, &n); err != nil {
				return err
			}
			u.IssuedAt = n.Time
		}
	}

	return nil
}

// GetExpirationTime implements the Claims interface.
func (u AuthClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return &jwt.NumericDate{u.ExpiresAt}, nil
}

// GetNotBefore implements the Claims interface.
func (u AuthClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

// GetIssuedAt implements the Claims interface.
func (u AuthClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return &jwt.NumericDate{u.IssuedAt}, nil
}

// GetAudience implements the Claims interface.
func (u AuthClaims) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil
}

// GetIssuer implements the Claims interface.
func (u AuthClaims) GetIssuer() (string, error) {
	return "", nil
}

// GetSubject implements the Claims interface.
func (u AuthClaims) GetSubject() (string, error) {
	return "", nil
}
