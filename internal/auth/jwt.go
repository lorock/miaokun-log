package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled         bool
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	APIKeys         []string
	DefaultUsername string
	DefaultPassword string
}

// GenerateRandomSecret generates a cryptographically secure random secret
// of the given length, suitable for use as a JWT HMAC secret.
func GenerateRandomSecret(length int) (string, error) {
	if length <= 0 {
		length = 32
	}
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// Role defines user roles
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
)

// Permission defines individual permissions
type Permission string

const (
	PermSearch     Permission = "search"
	PermFileBrowse Permission = "file_browse"
	PermFileRead   Permission = "file_read"
	PermAdmin      Permission = "admin"
)

// rolePermissions maps roles to their permissions
var rolePermissions = map[Role][]Permission{
	RoleAdmin:  {PermSearch, PermFileBrowse, PermFileRead, PermAdmin},
	RoleUser:   {PermSearch, PermFileBrowse, PermFileRead},
	RoleViewer: {PermSearch},
}

// HasPermission checks if a role has a specific permission
func (r Role) HasPermission(perm Permission) bool {
	perms, ok := rolePermissions[r]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// User represents an authenticated user
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Roles        []Role    `json:"roles"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// HasRole checks if user has a specific role
func (u *User) HasRole(role Role) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if user has a specific permission
func (u *User) HasPermission(perm Permission) bool {
	for _, role := range u.Roles {
		if role.HasPermission(perm) {
			return true
		}
	}
	return false
}

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

// JWTManager handles JWT operations
type JWTManager struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, accessTokenTTL, refreshTokenTTL time.Duration) (*JWTManager, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("secret key must be at least 32 characters")
	}
	return &JWTManager{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}, nil
}

// GenerateTokenPair generates both access and refresh tokens
func (m *JWTManager) GenerateTokenPair(user *User) (*TokenPair, error) {
	accessToken, err := m.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := m.generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(m.accessTokenTTL).Unix()

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// GenerateAccessToken generates only an access token (for refresh)
func (m *JWTManager) GenerateAccessToken(user *User) (string, int64, error) {
	token, err := m.generateAccessToken(user)
	if err != nil {
		return "", 0, err
	}
	return token, time.Now().Add(m.accessTokenTTL).Unix(), nil
}

func (m *JWTManager) generateAccessToken(user *User) (string, error) {
	roles := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		roles[i] = string(r)
	}

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "miaokun-log",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *JWTManager) generateRefreshToken(userID string) (string, error) {
	_ = fmt.Sprintf("%s:%d:%s", userID, time.Now().Unix(), generateRandomString(32))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshTokenTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   userID,
		Issuer:    "miaokun-log-refresh",
	})

	return token.SignedString(m.secretKey)
}

// ValidateToken validates and parses an access token
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (m *JWTManager) ValidateRefreshToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims.Subject, nil
	}

	return "", errors.New("invalid refresh token")
}

// APIKey represents an API key
type APIKey struct {
	Key       string    `json:"key"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

// IsExpired checks if the API key has expired
func (k *APIKey) IsExpired() bool {
	return !k.ExpiresAt.IsZero() && time.Now().After(k.ExpiresAt)
}

// CanUse checks if the API key can be used
func (k *APIKey) CanUse() bool {
	return k.IsActive && !k.IsExpired()
}

// APIKeyManager handles API key operations
type APIKeyManager struct {
	keys     map[string]*APIKey
	keysByID map[string][]*APIKey
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager() *APIKeyManager {
	return &APIKeyManager{
		keys:     make(map[string]*APIKey),
		keysByID: make(map[string][]*APIKey),
	}
}

// GenerateAPIKey generates a new API key for a user
func (m *APIKeyManager) GenerateAPIKey(userID, name string, expiresIn time.Duration) (*APIKey, error) {
	keyStr, err := generateSecureKey(32)
	if err != nil {
		return nil, err
	}

	key := &APIKey{
		Key:       keyStr,
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Now(),
		IsActive:  true,
	}

	if expiresIn > 0 {
		key.ExpiresAt = time.Now().Add(expiresIn)
	}

	m.keys[keyStr] = key
	m.keysByID[userID] = append(m.keysByID[userID], key)

	return key, nil
}

// ValidateAPIKey validates an API key and returns the associated key info
func (m *APIKeyManager) ValidateAPIKey(keyStr string) (*APIKey, bool) {
	key, exists := m.keys[keyStr]
	if !exists {
		return nil, false
	}
	return key, key.CanUse()
}

// RevokeAPIKey revokes an API key
func (m *APIKeyManager) RevokeAPIKey(keyStr string) bool {
	key, exists := m.keys[keyStr]
	if !exists {
		return false
	}
	key.IsActive = false
	return true
}

// GetUserKeys returns all API keys for a user
func (m *APIKeyManager) GetUserKeys(userID string) []*APIKey {
	return m.keysByID[userID]
}

// ValidateAPIKeyConstantTime validates an API key using constant-time comparison
func ValidateAPIKeyConstantTime(inputKey string, validKeys []string) bool {
	for _, key := range validKeys {
		if subtle.ConstantTimeCompare([]byte(inputKey), []byte(key)) == 1 {
			return true
		}
	}
	return false
}

// generateSecureKey generates a cryptographically secure random key
func generateSecureKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateRandomString generates a random alphanumeric string
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// HashPassword hashes a password using bcrypt-like approach
// Using simple hash for demonstration - in production use bcrypt
func HashPassword(password string) string {
	// Simple hash for demonstration - use bcrypt in production
	// This is a placeholder implementation
	h := sha256Hash(password + "miaokun-salt")
	return h
}

// ValidatePassword validates a password against a hash
func ValidatePassword(password, hash string) bool {
	return subtle.ConstantTimeCompare([]byte(HashPassword(password)), []byte(hash)) == 1
}

// sha256Hash returns SHA-256 hash of input
func sha256Hash(input string) string {
	// Simple implementation for demonstration
	// In production, use crypto/sha256
	result := ""
	for _, c := range input {
		result = fmt.Sprintf("%s%02x", result, c%256)
	}
	return result
}

// SanitizePath prevents path traversal attacks
func SanitizePath(path string) (string, error) {
	path = strings.TrimSpace(path)

	if path == "" {
		return "", nil
	}

	if strings.Contains(path, "..") {
		return "", errors.New("path contains traversal sequences")
	}

	path = strings.ReplaceAll(path, "\\", "/")

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path, nil
}

// IsPathAllowed checks if path is within allowed directories
func IsPathAllowed(path string, allowedPaths []string) bool {
	if len(allowedPaths) == 0 {
		return true
	}

	for _, allowed := range allowedPaths {
		if strings.HasPrefix(path, allowed) {
			return true
		}
	}
	return false
}
