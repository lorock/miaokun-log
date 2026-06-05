package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserStore manages users in memory
type UserStore struct {
	mu     sync.RWMutex
	users  map[string]*User
	byName map[string]*User
}

// NewUserStore creates a new user store with an optional default admin password.
// If defaultAdminPassword is empty, a secure random password is generated and returned
// as the second return value. This eliminates the hardcoded default credential risk.
func NewUserStore(defaultAdminPassword string) (*UserStore, string) {
	store := &UserStore{
		users:  make(map[string]*User),
		byName: make(map[string]*User),
	}

	adminPassword := defaultAdminPassword
	if adminPassword == "" {
		adminPassword = GenerateRandomPassword(24)
	}

	store.CreateUser(&User{
		ID:           "admin",
		Username:     "admin",
		PasswordHash: HashPasswordBcrypt(adminPassword),
		Roles:        []Role{RoleAdmin},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	return store, adminPassword
}

// GenerateRandomPassword generates a cryptographically secure random password
// of the given length using an alphanumeric + symbols alphabet.
func GenerateRandomPassword(length int) string {
	if length <= 0 {
		length = 24
	}
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	buf := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			// Fallback: use a hex-encoded random value if rand.Int fails
			randomBytes := make([]byte, (length/2)+1)
			_, _ = rand.Read(randomBytes)
			hexStr := hex.EncodeToString(randomBytes)
			if len(hexStr) > length {
				hexStr = hexStr[:length]
			}
			return hexStr
		}
		buf[i] = alphabet[n.Int64()]
	}
	return string(buf)
}

// CreateUser creates a new user
func (s *UserStore) CreateUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byName[user.Username]; exists {
		return fmt.Errorf("user %s already exists", user.Username)
	}

	if user.ID == "" {
		user.ID = generateUserID()
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()

	s.users[user.ID] = user
	s.byName[user.Username] = user
	return nil
}

// GetUserByID retrieves a user by ID
func (s *UserStore) GetUserByID(id string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.users[id]
	return user, exists
}

// GetUserByUsername retrieves a user by username
func (s *UserStore) GetUserByUsername(username string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.byName[username]
	return user, exists
}

// UpdateUser updates an existing user
func (s *UserStore) UpdateUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.users[user.ID]
	if !exists {
		return fmt.Errorf("user not found")
	}

	// Update fields
	if user.PasswordHash != "" {
		existing.PasswordHash = user.PasswordHash
	}
	if len(user.Roles) > 0 {
		existing.Roles = user.Roles
	}
	existing.UpdatedAt = time.Now()

	s.users[user.ID] = existing
	return nil
}

// DeleteUser deletes a user
func (s *UserStore) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	delete(s.users, id)
	delete(s.byName, user.Username)
	return nil
}

// ListUsers returns all users (without password hashes)
func (s *UserStore) ListUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, &User{
			ID:        user.ID,
			Username:  user.Username,
			Roles:     user.Roles,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}
	return users
}

// AuthenticateUser authenticates a user by username and password
func (s *UserStore) AuthenticateUser(username, password string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.byName[username]
	if !exists {
		return nil, false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, false
	}

	return user, true
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message,omitempty"`
	Data    *LoginData `json:"data,omitempty"`
	Error   *APIError  `json:"error,omitempty"`
}

// LoginData represents login success data
type LoginData struct {
	User  UserInfo  `json:"user"`
	Token TokenPair `json:"token"`
}

// UserInfo represents user information in responses
type UserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Roles     []Role `json:"roles"`
	CreatedAt string `json:"created_at"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents a token refresh response
type RefreshResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message,omitempty"`
	Data    *RefreshData `json:"data,omitempty"`
	Error   *APIError    `json:"error,omitempty"`
}

// RefreshData represents refresh success data
type RefreshData struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

// LogoutResponse represents a logout response
type LogoutResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	jwtManager *JWTManager
	userStore  *UserStore
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(jwtManager *JWTManager, userStore *UserStore) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
		userStore:  userStore,
	}
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST method required")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "MISSING_CREDENTIALS", "Username and password are required")
		return
	}

	user, ok := h.userStore.AuthenticateUser(req.Username, req.Password)
	if !ok {
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	tokens, err := h.jwtManager.GenerateTokenPair(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate tokens")
		return
	}

	userInfo := UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Roles:     user.Roles,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Success: true,
		Data: &LoginData{
			User:  userInfo,
			Token: *tokens,
		},
	})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST method required")
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "MISSING_TOKEN", "Refresh token is required")
		return
	}

	userID, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired refresh token")
		return
	}

	user, ok := h.userStore.GetUserByID(userID)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_NOT_FOUND", "User not found")
		return
	}

	accessToken, expiresAt, err := h.jwtManager.GenerateAccessToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate access token")
		return
	}

	writeJSON(w, http.StatusOK, RefreshResponse{
		Success: true,
		Data: &RefreshData{
			AccessToken: accessToken,
			ExpiresAt:   expiresAt,
		},
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST method required")
		return
	}

	// In a stateless JWT system, logout is handled client-side by discarding the token
	// Server-side token invalidation would require a token blacklist (not implemented here)
	writeJSON(w, http.StatusOK, LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "GET method required")
		return
	}

	claims := getClaimsFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	user, ok := h.userStore.GetUserByID(claims.UserID)
	if !ok {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	userInfo := UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Roles:     user.Roles,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    userInfo,
	})
}

// ListUsers returns all users (admin only)
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "GET method required")
		return
	}

	claims := getClaimsFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	// Check admin role
	isAdmin := false
	for _, role := range claims.Roles {
		if role == string(RoleAdmin) {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin role required")
		return
	}

	users := h.userStore.ListUsers()
	userInfos := make([]UserInfo, len(users))
	for i, u := range users {
		userInfos[i] = UserInfo{
			ID:        u.ID,
			Username:  u.Username,
			Roles:     u.Roles,
			CreatedAt: u.CreatedAt.Format(time.RFC3339),
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    userInfos,
	})
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": APIError{
			Code:    code,
			Message: message,
		},
	})
}

func generateUserID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// HashPasswordBcrypt hashes a password using bcrypt
func HashPasswordBcrypt(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}
