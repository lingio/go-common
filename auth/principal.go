package auth

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// Principal principal
type Principal struct {
	// email
	Email string `json:"email,omitempty"`

	// partner Id
	PartnerID string `json:"partnerId,omitempty"`

	// user Id
	UserID string `json:"userId,omitempty"`
}

// Validate validates this principal
func (m *Principal) MapData(claims *jwt.MapClaims) error {
	c := *claims
	if partnerID, ok := c["partnerId"]; ok {
		m.PartnerID = partnerID.(string)
	}
	if userID, ok := c["userId"]; ok {
		m.UserID = userID.(string)
	}
	if email, ok := c["email"]; ok {
		m.Email = email.(string)
	}
	return nil
}

// Validate validates this principal
func (m *Principal) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Principal) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Principal) UnmarshalBinary(b []byte) error {
	var res Principal
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}