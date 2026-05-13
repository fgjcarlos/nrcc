package service

import (
	"fmt"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and user management
type AuthService struct {
	jwtSecret string
	store     *store.JSONStore[model.CCUsers]
}

// NewAuthService creates a new auth service
func NewAuthService(jwtSecret string, store *store.JSONStore[model.CCUsers]) *AuthService {
	return &AuthService{
		jwtSecret: jwtSecret,
		store:     store,
	}
}

// GenerateToken generates a JWT token for a user
func (s *AuthService) GenerateToken(user *model.CCUser) (string, error) {
	now := time.Now()
	expiry := now.Add(24 * time.Hour)

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

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return &model.Claims{
		UserID:    (*claims)["userId"].(string),
		Username:  (*claims)["username"].(string),
		Role:      model.UserRole((*claims)["role"].(string)),
		ExpiresAt: int64((*claims)["exp"].(float64)),
		IssuedAt:  int64((*claims)["iat"].(float64)),
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

// HashPassword hashes a password using bcrypt
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
