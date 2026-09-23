package common

import (
	"crypto/rsa"
	"fmt"
	"testing"
	"testing/cryptotest"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthClaimsRoundTrip(t *testing.T) {
	cryptotest.SetGlobalRandom(t, 0)
	privateKey, _ := rsa.GenerateKey(nil, 1024)

	var (
		exp = time.Now().Add(1 * time.Minute)
		rbf = time.Now().Add(5 * time.Minute)
		iss = time.Now()
	)

	userToken := jwt.NewWithClaims(jwt.SigningMethodRS512, AuthClaims{
		PartnerID:     "pid",
		UserID:        "uid",
		DeviceID:      "did",
		Role:          "role",
		ExpiresAt:     exp,
		RefreshBefore: rbf,
		IssuedAt:      iss,
	})

	str, err := userToken.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	var (
		c AuthClaims
	)
	_, err = jwt.ParseWithClaims(str, &c, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &privateKey.PublicKey, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if a, b := "pid", c.PartnerID; a != b {
		t.Errorf("partner-id mismatch, expected: %v got: %v", a, b)
	}
	if a, b := "uid", c.UserID; a != b {
		t.Errorf("user-id mismatch, expected: %v got: %v", a, b)
	}
	if a, b := "did", c.DeviceID; a != b {
		t.Errorf("device-id mismatch, expected: %v got: %v", a, b)
	}
	if a, b := "role", c.Role; a != b {
		t.Errorf("role mismatch, expected: %v got: %v", a, b)
	}
	if a, b := exp.Unix(), c.ExpiresAt.Unix(); a != b {
		t.Errorf("exp mismatch, expected: %v got: %v", a, b)
	}
	if a, b := rbf.Unix(), c.RefreshBefore.Unix(); a != b {
		t.Errorf("rbf mismatch, expected: %v got: %v", a, b)
	}
	if a, b := iss.Unix(), c.IssuedAt.Unix(); a != b {
		t.Errorf("iss mismatch, expected: %v got: %v", a, b)
	}
}

func TestAuthClaimParseIssuedAtAsString(t *testing.T) {
	cryptotest.SetGlobalRandom(t, 0)
	privateKey, _ := rsa.GenerateKey(nil, 1024)

	x := `eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJwYXJ0bmVySWQiOiJwaWQiLCJ1c2VySWQiOiJ1aWQiLCJkZXZpY2VJZCI6ImRpZCIsInJvbGUiOiJyb2xlIiwiZXhwIjpudWxsLCJjcmVhdGVkIjoiMTc4NTk2NTMxOCJ9.xGMVXEkQXZxU20Lc2yFfceEhNNw07pHUQ6EVfaV_VLFxjb6__SwaEA8kYOGSYbiwrgqRw8psV1eJ22AW7GRLWByHduiDtgexJyOF11SwK8xlisSeY8c0hCor6RotMoXPn99DHHbm5dw7Xu5N0ATDYhiIAp2ocXQdMI1edxddlJA`

	var (
		c AuthClaims
	)
	_, err := jwt.ParseWithClaims(x, &c, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &privateKey.PublicKey, nil
	})
	if err == nil {
		t.Fatal("expected error while parsing expired token")
	}

	issuedAt := time.Unix(1785965318, 0)
	if issuedAt.Unix() != c.IssuedAt.Unix() {
		t.Errorf("issued-at as string not as expected: %v vs %v", issuedAt, c.IssuedAt)
	}
}

func TestAuthClaimParseIssuedAtAsTime(t *testing.T) {
	cryptotest.SetGlobalRandom(t, 0)
	privateKey, _ := rsa.GenerateKey(nil, 1024)

	x := `eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJwYXJ0bmVySWQiOiJwaWQiLCJ1c2VySWQiOiJ1aWQiLCJkZXZpY2VJZCI6ImRpZCIsInJvbGUiOiJyb2xlIiwiZXhwIjoxNzg1OTY2MDkzLCJyYmYiOjE3ODU5NjYzMzMsImNyZWF0ZWQiOjE3ODU5NjYwMzN9.YSadg7XRqJKWs5aLQLurhEge8Nl8H8QPXQSCsjVz36v3Es9O1-g0wGFfVJxzBnkyPlDF-s9wPTlBsPuUlLfp5ZPriX7j36cXj9akzrAFcABkJ3H8YpQF9yoWjVElmDG2q_AfLpyiuuHRdWC_LdsTDOPNVpbOnaR4HoUGUcsEU4w`

	var (
		c AuthClaims
	)
	_, err := jwt.ParseWithClaims(x, &c, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &privateKey.PublicKey, nil
	})
	if err == nil {
		t.Fatal("expected error while parsing expired token")
	}

	issuedAt := time.Unix(1785966033, 0)
	if issuedAt.Unix() != c.IssuedAt.Unix() {
		t.Errorf("issued-at as time not as expected: %v vs %v", issuedAt, c.IssuedAt)
	}
}
