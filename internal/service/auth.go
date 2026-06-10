package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessTokenLifetime  = 15 * time.Minute
	RefreshTokenLifetime = 7 * 24 * time.Hour
)

// AuthService handles authentication and user management
type AuthService struct {
	jwtSecret    string
	store        *store.JSONStore[model.CCUsers]
	sessionStore *store.JSONStore[model.RefreshSessions]
}

// NewAuthService creates a new auth service
func NewAuthService(jwtSecret string, userStore *store.JSONStore[model.CCUsers], sessionStore *store.JSONStore[model.RefreshSessions]) *AuthService {
	return &AuthService{
		jwtSecret:    jwtSecret,
		store:        userStore,
		sessionStore: sessionStore,
	}
}

// GenerateToken generates a short-lived JWT access token for a user.
func (s *AuthService) GenerateToken(user *model.CCUser) (string, error) {
	now := time.Now()
	expiry := now.Add(AccessTokenLifetime)

	claims := &model.Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		ExpiresAt: expiry.Unix(),
		IssuedAt:  now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":   claims.UserID,
		"username": claims.Username,
		"role":     claims.Role,
		"exp":      claims.ExpiresAt,
		"iat":      claims.IssuedAt,
	})

	return token.SignedString([]byte(s.jwtSecret))
}

// VerifyToken verifies a JWT token and returns claims
func (s *AuthService) VerifyToken(tokenStr string) (*model.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	mapClaims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	claims := *mapClaims

	userID, ok := claims["userId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token claims: userId")
	}
	username, ok := claims["username"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token claims: username")
	}
	role, ok := claims["role"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token claims: role")
	}
	// JSON numbers decode to float64 through jwt.MapClaims.
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid token claims: exp")
	}
	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid token claims: iat")
	}

	return &model.Claims{
		UserID:    userID,
		Username:  username,
		Role:      model.UserRole(role),
		ExpiresAt: int64(exp),
		IssuedAt:  int64(iat),
	}, nil
}

// GetUserByUsername retrieves a user by username
func (s *AuthService) GetUserByUsername(username string) *model.CCUser {
	users, err := s.store.Read()
	if err != nil {
		return nil
	}

	for _, u := range users.Users {
		if u.Username == username {
			return &u
		}
	}

	return nil
}

// VerifyPassword checks if the given password matches the hash
func (s *AuthService) VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// HashPassword hashes a password using bcrypt with explicit cost.
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	return string(hash), err
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(id string) *model.CCUser {
	users, err := s.store.Read()
	if err != nil {
		return nil
	}

	for _, u := range users.Users {
		if u.ID == id {
			return &u
		}
	}

	return nil
}

// GetAllUsers retrieves all users
func (s *AuthService) GetAllUsers() ([]model.CCUser, error) {
	users, err := s.store.Read()
	if err != nil {
		return nil, err
	}
	return users.Users, nil
}

// CreateUser creates a new user
func (s *AuthService) CreateUser(user *model.CCUser) error {
	users, err := s.store.Read()
	if err != nil {
		// If file doesn't exist, create new users object
		users = model.CCUsers{Users: []model.CCUser{}}
	}

	// Check if username already exists
	for _, u := range users.Users {
		if u.Username == user.Username {
			return fmt.Errorf("username already exists")
		}
	}

	users.Users = append(users.Users, *user)
	return s.store.Write(users)
}

// UpdateUser updates an existing user
func (s *AuthService) UpdateUser(user *model.CCUser) error {
	users, err := s.store.Read()
	if err != nil {
		return err
	}

	for i, u := range users.Users {
		if u.ID == user.ID {
			users.Users[i] = *user
			return s.store.Write(users)
		}
	}

	return fmt.Errorf("user not found")
}

// DeleteUser deletes a user by ID
func (s *AuthService) DeleteUser(id string) error {
	users, err := s.store.Read()
	if err != nil {
		return err
	}

	index := -1
	for i, u := range users.Users {
		if u.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("user not found")
	}

	users.Users = append(users.Users[:index], users.Users[index+1:]...)
	return s.store.Write(users)
}

// CreateRefreshSession creates a new refresh session and returns its opaque token.
func (s *AuthService) CreateRefreshSession(userID string) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	now := time.Now()
	session := model.RefreshSession{
		ID:        token,
		UserID:    userID,
		ExpiresAt: now.Add(RefreshTokenLifetime).Unix(),
		CreatedAt: now.Unix(),
	}

	sessions, _ := s.sessionStore.Read()
	sessions.Sessions = append(sessions.Sessions, session)
	if err := s.sessionStore.Write(sessions); err != nil {
		return "", fmt.Errorf("persist refresh session: %w", err)
	}

	return token, nil
}

// ValidateRefreshSession checks that a refresh token is valid, not expired, and not revoked.
func (s *AuthService) ValidateRefreshSession(token string) (*model.RefreshSession, error) {
	sessions, err := s.sessionStore.Read()
	if err != nil {
		return nil, fmt.Errorf("read sessions: %w", err)
	}

	for _, sess := range sessions.Sessions {
		if sess.ID == token {
			if sess.Revoked {
				return nil, fmt.Errorf("refresh token revoked")
			}
			if time.Now().Unix() > sess.ExpiresAt {
				return nil, fmt.Errorf("refresh token expired")
			}
			return &sess, nil
		}
	}

	return nil, fmt.Errorf("refresh token not found")
}

// RevokeRefreshSession marks a refresh session as revoked.
func (s *AuthService) RevokeRefreshSession(token string) error {
	sessions, err := s.sessionStore.Read()
	if err != nil {
		return fmt.Errorf("read sessions: %w", err)
	}

	for i, sess := range sessions.Sessions {
		if sess.ID == token {
			sessions.Sessions[i].Revoked = true
			return s.sessionStore.Write(sessions)
		}
	}

	return fmt.Errorf("refresh token not found")
}

// RevokeUserSessions revokes all refresh sessions for a user.
func (s *AuthService) RevokeUserSessions(userID string) error {
	sessions, err := s.sessionStore.Read()
	if err != nil {
		return nil
	}

	changed := false
	for i, sess := range sessions.Sessions {
		if sess.UserID == userID && !sess.Revoked {
			sessions.Sessions[i].Revoked = true
			changed = true
		}
	}

	if changed {
		return s.sessionStore.Write(sessions)
	}
	return nil
}

// PruneSessions removes expired and revoked sessions older than 24h.
func (s *AuthService) PruneSessions() {
	sessions, err := s.sessionStore.Read()
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	kept := make([]model.RefreshSession, 0, len(sessions.Sessions))
	for _, sess := range sessions.Sessions {
		if sess.Revoked && sess.ExpiresAt < cutoff {
			continue
		}
		if !sess.Revoked && time.Now().Unix() > sess.ExpiresAt && sess.ExpiresAt < cutoff {
			continue
		}
		kept = append(kept, sess)
	}

	if len(kept) < len(sessions.Sessions) {
		sessions.Sessions = kept
		_ = s.sessionStore.Write(sessions)
	}
}
