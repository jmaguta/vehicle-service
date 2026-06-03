package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
)

// Claims are the JWT claims issued by Supabase GoTrue.
type Claims struct {
	jwt.RegisteredClaims
	Email      string `json:"email"`
	Role       string `json:"role"`
	WorkshopID string `json:"workshop_id"`
	Tier       string `json:"tier"`
}

type ecJWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type jwksDoc struct {
	Keys []ecJWK `json:"keys"`
}

var (
	cachedECKeys   []*ecdsa.PublicKey
	cachedECKeysMu sync.RWMutex
)

func loadECKeys(url string) ([]*ecdsa.PublicKey, error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read jwks: %w", err)
	}
	var doc jwksDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse jwks: %w", err)
	}
	var keys []*ecdsa.PublicKey
	for _, k := range doc.Keys {
		if k.Kty != "EC" || k.Crv != "P-256" {
			continue
		}
		xb, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, fmt.Errorf("decode key x: %w", err)
		}
		yb, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, fmt.Errorf("decode key y: %w", err)
		}
		keys = append(keys, &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(xb),
			Y:     new(big.Int).SetBytes(yb),
		})
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no EC P-256 keys found in JWKS")
	}
	return keys, nil
}

func getECKeys() ([]*ecdsa.PublicKey, error) {
	cachedECKeysMu.RLock()
	keys := cachedECKeys
	cachedECKeysMu.RUnlock()
	if keys != nil {
		return keys, nil
	}
	cachedECKeysMu.Lock()
	defer cachedECKeysMu.Unlock()
	if cachedECKeys != nil {
		return cachedECKeys, nil
	}
	url := os.Getenv("SUPABASE_JWKS_URL")
	if url == "" {
		return nil, fmt.Errorf("SUPABASE_JWKS_URL not set")
	}
	keys, err := loadECKeys(url)
	if err != nil {
		return nil, err
	}
	cachedECKeys = keys
	return keys, nil
}

// ValidateJWT parses and validates a Supabase GoTrue JWT.
// Uses EC P-256 via JWKS when SUPABASE_JWKS_URL is set (Supabase Cloud),
// falling back to HS256 via SUPABASE_JWT_SECRET for self-hosted / local dev.
func ValidateJWT(tokenStr string) (*Claims, error) {
	if os.Getenv("SUPABASE_JWKS_URL") != "" {
		return validateEC(tokenStr)
	}
	return validateHMAC(tokenStr)
}

func validateEC(tokenStr string) (*Claims, error) {
	keys, err := getECKeys()
	if err != nil {
		return nil, fmt.Errorf("load jwks: %w", err)
	}
	var lastErr error
	for _, key := range keys {
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return key, nil
		})
		if err != nil {
			lastErr = err
			continue
		}
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			return claims, nil
		}
		lastErr = fmt.Errorf("invalid token")
	}
	return nil, fmt.Errorf("parse jwt: %w", lastErr)
}

func validateHMAC(tokenStr string) (*Claims, error) {
	secret := os.Getenv("SUPABASE_JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("SUPABASE_JWT_SECRET not set")
	}
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// BearerToken extracts the token string from an "Authorization: Bearer <token>" header value.
func BearerToken(header string) (string, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", fmt.Errorf("malformed Authorization header")
	}
	return strings.TrimSpace(parts[1]), nil
}
