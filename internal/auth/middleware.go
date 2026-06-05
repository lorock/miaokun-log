package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// ContextKey type for context keys
type ContextKey string

const (
	// ClaimsKey is the context key for JWT claims
	ClaimsKey ContextKey = "claims"
	// UserKey is the context key for authenticated user
	UserKey ContextKey = "user"
	// PermissionsKey is the context key for user permissions
	PermissionsKey ContextKey = "permissions"
)

// contextKey is used for context values
type contextKey string

// JWTAuthMiddleware provides JWT authentication middleware
type JWTAuthMiddleware struct {
	jwtManager *JWTManager
	apiKeys    *APIKeyManager
}

// NewJWTAuthMiddleware creates a new JWT auth middleware
func NewJWTAuthMiddleware(jwtManager *JWTManager, apiKeys *APIKeyManager) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		jwtManager: jwtManager,
		apiKeys:    apiKeys,
	}
}

// Authenticate returns a middleware that authenticates requests
func (m *JWTAuthMiddleware) Authenticate() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := m.extractAndValidateToken(r)
			if err != nil {
				// Try API key authentication
				apiKey, valid := m.tryAPIKeyAuth(r)
				if !valid {
					writeUnauthorized(w, "AUTHENTICATION_REQUIRED", "请先登录后再操作")
					return
				}
				// Set API key user in context
				ctx := context.WithValue(r.Context(), contextKey(UserKey), &User{ID: apiKey.UserID})
				ctx = context.WithValue(ctx, contextKey("api_key"), apiKey)
				r = r.WithContext(ctx)
			} else {
				// Set JWT user in context
				ctx := context.WithValue(r.Context(), ClaimsKey, claims)
				ctx = context.WithValue(ctx, UserKey, &User{
					ID:       claims.UserID,
					Username: claims.Username,
					Roles:    toRoles(claims.Roles),
				})
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission returns a middleware that requires a specific permission
func (m *JWTAuthMiddleware) RequirePermission(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := getUserFromContext(r)
			if user == nil {
				writeForbidden(w, "NOT_AUTHENTICATED", "请先登录后再操作")
				return
			}

			if !user.HasPermission(perm) {
				writeForbidden(w, "PERMISSION_DENIED", "您没有执行此操作的权限")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns a middleware that requires a specific role
func (m *JWTAuthMiddleware) RequireRole(role Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := getUserFromContext(r)
			if user == nil {
				writeForbidden(w, "NOT_AUTHENTICATED", "请先登录后再操作")
				return
			}

			if !user.HasRole(role) {
				writeForbidden(w, "ROLE_REQUIRED", "此操作需要 "+string(role)+" 角色权限")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractAndValidateToken extracts and validates the JWT token from request
func (m *JWTAuthMiddleware) extractAndValidateToken(r *http.Request) (*Claims, error) {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return m.jwtManager.ValidateToken(parts[1])
		}
	}

	// Try query parameter as fallback
	token := r.URL.Query().Get("access_token")
	if token != "" {
		return m.jwtManager.ValidateToken(token)
	}

	return nil, http.ErrNotSupported
}

// tryAPIKeyAuth attempts to authenticate using API key
func (m *JWTAuthMiddleware) tryAPIKeyAuth(r *http.Request) (*APIKey, bool) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}

	if apiKey == "" {
		return nil, false
	}

	return m.apiKeys.ValidateAPIKey(apiKey)
}

// getUserFromContext extracts user from request context
func getUserFromContext(r *http.Request) *User {
	user, ok := r.Context().Value(UserKey).(*User)
	if !ok {
		return nil
	}
	return user
}

// getClaimsFromContext extracts claims from request context
func getClaimsFromContext(r *http.Request) *Claims {
	claims, ok := r.Context().Value(ClaimsKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// toRoles converts string slice to Role slice
func toRoles(roles []string) []Role {
	result := make([]Role, len(roles))
	for i, r := range roles {
		result[i] = Role(r)
	}
	return result
}

// writeUnauthorized writes an unauthorized response
func writeUnauthorized(w http.ResponseWriter, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// writeForbidden writes a forbidden response
func writeForbidden(w http.ResponseWriter, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// OptionalAuth provides optional authentication - doesn't fail if no auth
func OptionalAuth(jwtManager *JWTManager, apiKeys *APIKeyManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try JWT auth
			claims, err := validateToken(r, jwtManager)
			if err == nil && claims != nil {
				ctx := context.WithValue(r.Context(), ClaimsKey, claims)
				ctx = context.WithValue(ctx, UserKey, &User{
					ID:       claims.UserID,
					Username: claims.Username,
					Roles:    toRoles(claims.Roles),
				})
				r = r.WithContext(ctx)
			} else {
				// Try API key
				apiKey := getAPIKey(r, apiKeys)
				if apiKey != nil {
					ctx := context.WithValue(r.Context(), UserKey, &User{ID: apiKey.UserID})
					ctx = context.WithValue(ctx, "api_key", apiKey)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validateToken validates JWT token
func validateToken(r *http.Request, jwtManager *JWTManager) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return jwtManager.ValidateToken(parts[1])
		}
	}

	token := r.URL.Query().Get("access_token")
	if token != "" {
		return jwtManager.ValidateToken(token)
	}

	return nil, http.ErrNotSupported
}

// getAPIKey extracts and validates API key
func getAPIKey(r *http.Request, apiKeys *APIKeyManager) *APIKey {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}

	if apiKey == "" {
		return nil
	}

	key, valid := apiKeys.ValidateAPIKey(apiKey)
	if !valid {
		return nil
	}
	return key
}
