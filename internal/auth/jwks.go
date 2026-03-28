// Package auth provides JWT validation and Permission Service integration
// for the galaxis game backend.
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserContext holds the validated claims extracted from a Keycloak JWT.
type UserContext struct {
	UserID       uuid.UUID
	Email        string
	RealmRoles   []string // e.g. "player", "platform-admin"
	GalaxisRoles []string // from resource_access.galaxis.roles: "player", "game-admin", "spectator"
}

func (u *UserContext) HasRealmRole(role string) bool {
	for _, r := range u.RealmRoles {
		if r == role {
			return true
		}
	}
	return false
}

func (u *UserContext) HasGalaxisRole(role string) bool {
	for _, r := range u.GalaxisRoles {
		if r == role {
			return true
		}
	}
	return false
}

type ctxKey struct{}

// ValidateFunc validates a raw JWT and returns the UserContext.
type ValidateFunc func(ctx context.Context, tokenStr string) (*UserContext, error)

// NewJWKSValidator builds a ValidateFunc backed by Keycloak JWKS with a 5-minute cache.
func NewJWKSValidator(jwksURL, issuer string) ValidateFunc {
	var (
		mu      sync.RWMutex
		keySet  map[string]*rsa.PublicKey
		fetched time.Time
	)

	fetchKeys := func() error {
		resp, err := http.Get(jwksURL) //nolint:gosec
		if err != nil {
			return fmt.Errorf("fetch jwks: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var raw struct {
			Keys []json.RawMessage `json:"keys"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return fmt.Errorf("decode jwks: %w", err)
		}

		keys := make(map[string]*rsa.PublicKey, len(raw.Keys))
		for _, k := range raw.Keys {
			var key struct {
				Kid string `json:"kid"`
				Kty string `json:"kty"`
				N   string `json:"n"`
				E   string `json:"e"`
			}
			if err := json.Unmarshal(k, &key); err != nil || key.Kty != "RSA" {
				continue
			}
			pub, err := rsaPublicKeyFromJWK(key.N, key.E)
			if err == nil {
				keys[key.Kid] = pub
			}
		}

		mu.Lock()
		keySet = keys
		fetched = time.Now()
		mu.Unlock()
		return nil
	}

	return func(_ context.Context, tokenStr string) (*UserContext, error) {
		mu.RLock()
		stale := time.Since(fetched) > 5*time.Minute
		mu.RUnlock()
		if stale {
			if err := fetchKeys(); err != nil {
				return nil, fmt.Errorf("jwks refresh: %w", err)
			}
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			kid, _ := t.Header["kid"].(string)
			mu.RLock()
			key, ok := keySet[kid]
			mu.RUnlock()
			if !ok {
				return nil, fmt.Errorf("unknown kid: %s", kid)
			}
			return key, nil
		},
			jwt.WithIssuer(issuer),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !token.Valid {
			return nil, fmt.Errorf("invalid token: %w", err)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid claims type")
		}

		sub, _ := claims["sub"].(string)
		userID, err := uuid.Parse(sub)
		if err != nil {
			return nil, fmt.Errorf("invalid sub: %w", err)
		}

		email, _ := claims["email"].(string)
		realmRoles := extractStringSlice(claims, "realm_access", "roles")
		galaxisRoles := extractClientRoles(claims, "galaxis")

		return &UserContext{
			UserID:       userID,
			Email:        email,
			RealmRoles:   realmRoles,
			GalaxisRoles: galaxisRoles,
		}, nil
	}
}

// FromContext retrieves the UserContext stored by the Authenticate middleware.
func FromContext(ctx context.Context) *UserContext {
	uc, _ := ctx.Value(ctxKey{}).(*UserContext)
	return uc
}

func toContext(ctx context.Context, uc *UserContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, uc)
}

// rsaPublicKeyFromJWK constructs an RSA public key from base64url-encoded n and e.
func rsaPublicKeyFromJWK(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() {
		return nil, fmt.Errorf("exponent too large")
	}
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

func extractStringSlice(claims jwt.MapClaims, keys ...string) []string {
	var cur interface{} = map[string]interface{}(claims)
	for _, k := range keys {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = m[k]
	}
	raw, ok := cur.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func extractClientRoles(claims jwt.MapClaims, clientID string) []string {
	ra, ok := claims["resource_access"].(map[string]interface{})
	if !ok {
		return nil
	}
	client, ok := ra[clientID].(map[string]interface{})
	if !ok {
		return nil
	}
	roles, ok := client["roles"].([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(roles))
	for _, r := range roles {
		if s, ok := r.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
