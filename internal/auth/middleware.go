package auth

import (
	"net/http"
	"strings"
)

// Authenticate is a chi middleware that validates the Bearer token and stores
// the UserContext in the request context.
//
// When validate is nil (KEYCLOAK_JWKS_URL not configured) the middleware is a no-op
// — useful for local dev without a running Keycloak.
func Authenticate(validate ValidateFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if validate == nil {
				next.ServeHTTP(w, r)
				return
			}

			tokenStr := tokenFromRequest(r)
			if tokenStr == "" {
				http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}

			uc, err := validate(r.Context(), tokenStr)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(toContext(r.Context(), uc)))
		})
	}
}

// RequirePlayer rejects requests from users without the galaxis "player" role.
func RequirePlayer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uc := FromContext(r.Context())
		if uc == nil || (!uc.HasGalaxisRole("player") && !uc.HasGalaxisRole("game-admin") && !uc.HasRealmRole("platform-admin")) {
			http.Error(w, "forbidden: galaxis player role required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireGameAdmin rejects requests from users without the galaxis "game-admin" role.
func RequireGameAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uc := FromContext(r.Context())
		if uc == nil || (!uc.HasGalaxisRole("game-admin") && !uc.HasRealmRole("platform-admin")) {
			http.Error(w, "forbidden: game-admin role required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// tokenFromRequest extracts the Bearer token from the Authorization header.
// Falls back to "token" query parameter for WebSocket upgrade requests.
func tokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	// WebSocket fallback: ?token=<jwt>
	// Only accepted on WS upgrade paths to avoid token in regular request logs.
	if r.Header.Get("Upgrade") == "websocket" {
		return r.URL.Query().Get("token")
	}
	return ""
}
