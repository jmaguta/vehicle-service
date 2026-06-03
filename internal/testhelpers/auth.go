package testhelpers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestJWTSecret is the signing secret used for all test tokens.
// Tests should set SUPABASE_JWT_SECRET to this value via t.Setenv.
const TestJWTSecret = "test-secret-for-unit-tests"

// MakeToken returns a signed HS256 JWT with the given subject, role, and workshop_id.
func MakeToken(subject, role, workshopID string) string {
	claims := jwt.MapClaims{
		"sub":         subject,
		"exp":         time.Now().Add(time.Hour).Unix(),
		"role":        role,
		"workshop_id": workshopID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(TestJWTSecret))
	return s
}
