package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessTokenLifetime  = 15 * time.Minute
	RefreshTokenLifetime = 7 * 24 * time.Hour
)

// hashRefreshToken returns the SHA-256 hex digest of a refresh token. Used
// to make the on-disk id of a refresh session unguessable from the file
// alone — the raw token is what the client holds, only its digest is
// persisted (#669). SHA-256 (no salt) is sufficient because the input is
// 256 bits of CSPRNG output; there is no low-entropy material to brute force.
func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

var (
	ErrAlreadyConfigured     = errors.New("system already configured")
	ErrUsernameExists        = errors.New("username already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrCannotDeleteLastAdmin = errors.New("cannot delete the last admin user")
	ErrCannotDemoteLastAdmin = errors.New("cannot demote the last admin user")
	ErrInvalidUser           = errors.New("invalid user")
)

// AuthService handles authentication and user management
type AuthService struct {
	jwtSecret    string
	store        *store.JSONStore[model.CCUsers]
	sessionStore *store.JSONStore[model.RefreshSessions]
	mfaStore     *store.JSONStore[model.MfaStore]
	credentialMu sync.Mutex
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
		"userId":            claims.UserID,
		"username":          claims.Username,
		"role":              claims.Role,
		"exp":               claims.ExpiresAt,
		"iat":               claims.IssuedAt,
		"credentialVersion": credentialVersion(user),
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

	version, ok := claims["credentialVersion"].(string)
	if _, present := claims["credentialVersion"]; present && !ok {
		return nil, ErrCredentialsChanged
	}
	user := s.GetUserByID(userID)
	if user == nil || credentialVersion(user) != version {
		return nil, ErrCredentialsChanged
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
			user, _ := s.withMfaVersion(u)
			return user
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
	user, _ := s.getUserByID(id)
	return user
}

func (s *AuthService) getUserByID(id string) (*model.CCUser, error) {
	users, err := s.store.Read()
	if err != nil {
		return nil, err
	}
	for _, u := range users.Users {
		if u.ID == id {
			return s.withMfaVersion(u)
		}
	}
	return nil, ErrUserNotFound
}

// GetAllUsers retrieves all users
func (s *AuthService) GetAllUsers() ([]model.CCUser, error) {
	users, err := s.store.Read()
	if err != nil {
		return nil, err
	}
	return users.Users, nil
}

// CountAdmins returns the number of users with the admin role. Used by
// the security posture endpoint (issue #676 item 2) to compute the
// MFA-enrollment fraction. Errors are swallowed (returning 0) so the
// posture endpoint stays servable if the user store is unavailable
// during bootstrap; the dashboard then renders the chip as "unknown"
// via the count of zero.
func (s *AuthService) CountAdmins() int {
	users, err := s.GetAllUsers()
	if err != nil {
		return 0
	}
	n := 0
	for _, u := range users {
		if u.Role == model.RoleAdmin {
			n++
		}
	}
	return n
}

// CountActiveRefreshSessions returns the number of refresh sessions
// that are unrevoked, unexpired and match current credentials. Used by
// the security posture endpoint (issue #676 item 2). Same swallow-
// on-error pattern as CountAdmins.
func (s *AuthService) CountActiveRefreshSessions() int {
	sessions, err := s.sessionStore.Read()
	if err != nil {
		return 0
	}
	now := time.Now().Unix()
	users, err := s.store.Read()
	if err != nil {
		return 0
	}
	mfaVersions, err := s.mfaVersions()
	if err != nil {
		return 0
	}
	versions := make(map[string]string, len(users.Users))
	for _, user := range users.Users {
		user.MfaVersion = mfaVersions[user.ID]
		versions[user.ID] = credentialVersion(&user)
	}
	n := 0
	for _, sess := range sessions.Sessions {
		if sess.Revoked {
			continue
		}
		if sess.ExpiresAt <= now {
			continue
		}
		if version, ok := versions[sess.UserID]; ok && version == sess.CredentialVersion {
			n++
		}
	}
	return n
}

// CreateUser creates a new user
func (s *AuthService) CreateUser(user *model.CCUser) error {
	if user == nil {
		return ErrInvalidUser
	}

	return s.store.Update(func(users *model.CCUsers) error {
		for _, current := range users.Users {
			if current.Username == user.Username {
				return ErrUsernameExists
			}
		}

		users.Users = append(users.Users, *user)
		return nil
	})
}

// BootstrapFirstAdmin atomically creates the initial administrative user.
func (s *AuthService) BootstrapFirstAdmin(user *model.CCUser) error {
	return s.store.Update(func(users *model.CCUsers) error {
		if len(users.Users) != 0 {
			return ErrAlreadyConfigured
		}
		if user == nil || user.ID == "" || user.Username == "" || user.PasswordHash == "" {
			return ErrInvalidUser
		}

		firstAdmin := *user
		firstAdmin.Role = model.RoleAdmin
		users.Users = append(users.Users, firstAdmin)
		return nil
	})
}

// UpdateUser atomically locates a user and applies a field-level mutation.
func (s *AuthService) UpdateUser(id string, mutate func(*model.CCUser) error) error {
	if mutate == nil {
		return errors.New("nil user mutation")
	}

	return s.store.Update(func(users *model.CCUsers) error {
		for i := range users.Users {
			if users.Users[i].ID == id {
				return mutate(&users.Users[i])
			}
		}

		return ErrUserNotFound
	})
}

// DeleteUser deletes a user by ID
func (s *AuthService) DeleteUser(id string) error {
	_, err := s.deleteUser(id, false)
	return err
}

// DeleteUserWithPolicy atomically validates the last-admin invariant and removes a user.
func (s *AuthService) DeleteUserWithPolicy(id string) (*model.CCUser, error) {
	return s.deleteUser(id, true)
}

func (s *AuthService) deleteUser(id string, preserveLastAdmin bool) (*model.CCUser, error) {
	var deleted model.CCUser
	err := s.store.Update(func(users *model.CCUsers) error {
		index := -1
		for i := range users.Users {
			if users.Users[i].ID == id {
				index = i
				break
			}
		}
		if index == -1 {
			return ErrUserNotFound
		}

		if preserveLastAdmin && users.Users[index].Role == model.RoleAdmin {
			adminCount := 0
			for _, user := range users.Users {
				if user.Role == model.RoleAdmin {
					adminCount++
				}
			}
			if adminCount <= 1 {
				return ErrCannotDeleteLastAdmin
			}
		}

		deleted = users.Users[index]
		users.Users = append(users.Users[:index], users.Users[index+1:]...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &deleted, nil
}

// UpdateUserRole atomically changes a role while preserving at least one admin.
func (s *AuthService) UpdateUserRole(id string, role model.UserRole, updatedAt string) (*model.CCUser, error) {
	var updated model.CCUser
	err := s.store.Update(func(users *model.CCUsers) error {
		index := -1
		for i := range users.Users {
			if users.Users[i].ID == id {
				index = i
				break
			}
		}
		if index == -1 {
			return ErrUserNotFound
		}

		if role == model.RoleViewer && users.Users[index].Role == model.RoleAdmin {
			adminCount := 0
			for _, user := range users.Users {
				if user.Role == model.RoleAdmin {
					adminCount++
				}
			}
			if adminCount <= 1 {
				return ErrCannotDemoteLastAdmin
			}
		}

		users.Users[index].Role = role
		users.Users[index].UpdatedAt = updatedAt
		updated = users.Users[index]
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// CreateRefreshSession creates a new refresh session and returns its opaque token.
//
// The returned token is the only secret the client ever sees. The session
// record persisted to disk stores sha256(token) as its id, not the token
// itself (#669), so a leaked sessions.json does not yield usable bearer
// credentials.
func (s *AuthService) CreateRefreshSession(userID string) (string, error) {
	user := s.GetUserByID(userID)
	if user == nil {
		return "", ErrUserNotFound
	}
	return s.CreateRefreshSessionForUser(user)
}

// CreateRefreshSessionForUser binds the session to the credential snapshot
// that was authenticated, not to whatever credentials exist at issuance time.
func (s *AuthService) CreateRefreshSessionForUser(user *model.CCUser) (string, error) {
	return s.replaceRefreshSession(user, nil)
}

func newRefreshSession(user *model.CCUser) (string, model.RefreshSession, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", model.RefreshSession{}, fmt.Errorf("generate refresh token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	now := time.Now()
	return token, model.RefreshSession{
		ID: hashRefreshToken(token), UserID: user.ID,
		ExpiresAt: now.Add(RefreshTokenLifetime).Unix(), CreatedAt: now.Unix(),
		CredentialVersion: credentialVersion(user),
	}, nil
}

func (s *AuthService) replaceRefreshSession(user *model.CCUser, old *model.RefreshSession) (string, error) {
	token, session, err := newRefreshSession(user)
	if err != nil {
		return "", err
	}
	err = s.sessionStore.Update(func(sessions *model.RefreshSessions) error {
		if old != nil {
			current, err := findValidRefreshSession(sessions.Sessions, old.ID)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidRefresh, err)
			}
			current.Revoked, current.RevokedAt = true, time.Now().Unix()
		}
		sessions.Sessions = append(sessions.Sessions, session)
		return nil
	})
	return token, err
}

// RotateRefreshSession keeps the credential snapshot fixed until the ledger
// update commits. Validation is repeated inside Update for logout and replay.
func (s *AuthService) RotateRefreshSession(token string) (*model.CCUser, string, error) {
	s.credentialMu.Lock()
	defer s.credentialMu.Unlock()
	session, user, err := s.validateRefreshSession(token)
	if errors.Is(err, ErrUserNotFound) {
		// Keep the first deleted-user response, but retire the cookie so
		// subsequent attempts take the normal invalid-token rate-limit path.
		if revokeErr := s.RevokeRefreshSession(token); revokeErr != nil {
			return nil, "", revokeErr
		}
		return nil, "", err
	}
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrInvalidRefresh, err)
	}
	replacement, err := s.replaceRefreshSession(user, session)
	return user, replacement, err
}

// CreateRefreshSessionForTest writes a caller-supplied RefreshSession
// verbatim into the session ledger. Used by handler tests that need to
// seed expired/revoked sessions without going through the full
// token-minting flow. Not exposed outside the handler test packages.
func (s *AuthService) CreateRefreshSessionForTest(session model.RefreshSession) error {
	return s.sessionStore.Update(func(sessions *model.RefreshSessions) error {
		sessions.Sessions = append(sessions.Sessions, session)
		return nil
	})
}

// ValidateRefreshSession checks that a refresh token is valid, not expired, and not revoked.
func (s *AuthService) ValidateRefreshSession(token string) (*model.RefreshSession, error) {
	session, _, err := s.validateRefreshSession(token)
	return session, err
}

func (s *AuthService) validateRefreshSession(token string) (*model.RefreshSession, *model.CCUser, error) {
	sessions, err := s.sessionStore.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("read sessions: %w", err)
	}
	session, err := findValidRefreshSession(sessions.Sessions, hashRefreshToken(token))
	if err != nil {
		return nil, nil, err
	}
	user, err := s.getUserByID(session.UserID)
	if err != nil {
		return nil, nil, err
	}
	if credentialVersion(user) != session.CredentialVersion {
		return nil, nil, ErrCredentialsChanged
	}
	return session, user, nil
}

func findValidRefreshSession(sessions []model.RefreshSession, tokenHash string) (*model.RefreshSession, error) {
	for i := range sessions {
		sess := &sessions[i]
		if subtle.ConstantTimeCompare([]byte(sess.ID), []byte(tokenHash)) != 1 {
			continue
		}
		if sess.Revoked {
			return nil, fmt.Errorf("refresh token revoked")
		}
		if time.Now().Unix() > sess.ExpiresAt {
			return nil, fmt.Errorf("refresh token expired")
		}
		return sess, nil
	}
	return nil, fmt.Errorf("refresh token not found")
}

// RevokeRefreshSession marks a refresh session as revoked.
func (s *AuthService) RevokeRefreshSession(token string) error {
	return s.sessionStore.Update(func(sessions *model.RefreshSessions) error {
		tokenHash := hashRefreshToken(token)
		for i := range sessions.Sessions {
			if subtle.ConstantTimeCompare([]byte(sessions.Sessions[i].ID), []byte(tokenHash)) != 1 {
				continue
			}
			if !sessions.Sessions[i].Revoked {
				sessions.Sessions[i].Revoked = true
				sessions.Sessions[i].RevokedAt = time.Now().Unix()
			}
			return nil
		}

		return fmt.Errorf("refresh token not found")
	})
}

// RevokeUserSessions revokes all refresh sessions for a user.
func (s *AuthService) RevokeUserSessions(userID string) error {
	return s.sessionStore.Update(func(sessions *model.RefreshSessions) error {
		revokedAt := time.Now().Unix()
		for i := range sessions.Sessions {
			session := &sessions.Sessions[i]
			if session.UserID == userID && !session.Revoked {
				session.Revoked = true
				session.RevokedAt = revokedAt
			}
		}
		return nil
	})
}

// PruneSessions removes sessions expired or revoked more than 24h ago.
func (s *AuthService) PruneSessions() error {
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	return s.sessionStore.Update(func(sessions *model.RefreshSessions) error {
		kept := make([]model.RefreshSession, 0, len(sessions.Sessions))
		for _, session := range sessions.Sessions {
			// Legacy revocations have no timestamp and are already invalid.
			oldRevocation := session.Revoked && (session.RevokedAt == 0 || session.RevokedAt < cutoff)
			if session.ExpiresAt < cutoff || oldRevocation {
				continue
			}
			kept = append(kept, session)
		}
		sessions.Sessions = kept
		return nil
	})
}
