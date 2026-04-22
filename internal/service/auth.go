package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/security"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const SessionCookieName = "nrcc_session"

type UserActionError struct {
	Status  int
	Code    string
	Message string
}

func (e *UserActionError) Error() string {
	return e.Message
}

func IsUserActionError(err error) (*UserActionError, bool) {
	var actionErr *UserActionError
	if !errors.As(err, &actionErr) {
		return nil, false
	}
	return actionErr, true
}

type AuthService struct {
	dataDir   string
	db        *sql.DB
	session   *security.SessionManager
	cookieTTL time.Duration
	mu        sync.Mutex
	attempts  map[string][]time.Time
}

func NewAuthService(dataDir string) (*AuthService, error) {
	dbPath := filepath.Join(dataDir, "nrcc.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err := initAuthSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	session, err := security.NewSessionManager(filepath.Join(dataDir, ".session-secret"))
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &AuthService{
		dataDir:   dataDir,
		db:        db,
		session:   session,
		cookieTTL: 24 * time.Hour,
		attempts:  make(map[string][]time.Time),
	}, nil
}

func (s *AuthService) GetDB() *sql.DB {
	return s.db
}

func (s *AuthService) RegisterInitial(username, password string) (*model.UserPublic, string, error) {
	username = strings.TrimSpace(username)
	if err := validateCredentials(username, password); err != nil {
		return nil, "", err
	}

	hasUsers, err := s.HasUsers()
	if err != nil {
		return nil, "", err
	}
	if hasUsers {
		return nil, "", fmt.Errorf("initial user has already been created")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	record := model.UserRecord{
		ID:           fmt.Sprintf("usr_%d", time.Now().UnixNano()),
		Username:     username,
		PasswordHash: string(hash),
		Role:         model.RoleAdmin,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.insertUser(record); err != nil {
		return nil, "", err
	}

	token, err := s.issueToken(record)
	if err != nil {
		return nil, "", err
	}

	s.logAudit("auth.bootstrap", username, "initial administrator created")

	user := publicUser(record)
	return &user, token, nil
}

func (s *AuthService) Login(username, password, clientAddr string) (*model.UserPublic, string, error) {
	username = strings.TrimSpace(username)
	if limited, retryAfter := s.loginLimited(username, clientAddr); limited {
		return nil, "", fmt.Errorf("too many login attempts, retry in %s", retryAfter.Round(time.Second))
	}

	user, err := s.findUserByUsername(username)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		s.recordLoginFailure(username, clientAddr)
		s.logAudit("auth.login_failed", username, "unknown username")
		return nil, "", fmt.Errorf("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.recordLoginFailure(username, clientAddr)
		s.logAudit("auth.login_failed", username, "invalid password")
		return nil, "", fmt.Errorf("invalid username or password")
	}

	s.clearLoginFailures(username, clientAddr)
	token, err := s.issueToken(*user)
	if err != nil {
		return nil, "", err
	}

	s.logAudit("auth.login_succeeded", username, "login succeeded")

	public := publicUser(*user)
	return &public, token, nil
}

func (s *AuthService) VerifyToken(token string) (*model.SessionClaims, error) {
	claims, err := s.session.Verify(token)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(claims.SID) == "" {
		return nil, fmt.Errorf("invalid session")
	}

	if err := s.deleteExpiredSessions(); err != nil {
		return nil, err
	}

	var count int
	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM sessions
		WHERE id = ? AND user_id = ? AND expires_at = ?
	`, claims.SID, claims.Sub, time.Unix(claims.Exp, 0).UTC().Format(time.RFC3339)).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("verify session: %w", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("invalid session")
	}

	return claims, nil
}

func (s *AuthService) HasUsers() (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return count > 0, nil
}

func (s *AuthService) SessionTTL() time.Duration {
	return s.cookieTTL
}

func (s *AuthService) CSRFToken(sessionToken string) string {
	return s.session.CSRFToken(sessionToken)
}

func (s *AuthService) VerifyCSRF(sessionToken, token string) bool {
	return s.session.VerifyCSRF(sessionToken, token)
}

func (s *AuthService) LogAudit(eventType, username, detail string) {
	s.logAudit(eventType, username, detail)
}

func (s *AuthService) RevokeToken(token string) error {
	claims, err := s.session.Verify(token)
	if err != nil {
		return err
	}
	if strings.TrimSpace(claims.SID) == "" {
		return fmt.Errorf("invalid session")
	}

	if _, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, claims.SID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *AuthService) Close() error {
	return s.db.Close()
}

func (s *AuthService) FindPublicUserByID(id string) (*model.UserPublic, error) {
	row := s.db.QueryRow(`
		SELECT id, username, role, created_at
		FROM users
		WHERE id = ?
	`, id)

	var user model.UserPublic
	if err := row.Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return &user, nil
}

func (s *AuthService) ListUsers() ([]model.UserPublic, error) {
	rows, err := s.db.Query(`
		SELECT id, username, role, created_at
		FROM users
		ORDER BY username COLLATE NOCASE ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]model.UserPublic, 0)
	for rows.Next() {
		var user model.UserPublic
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

func (s *AuthService) CreateUser(username, password string, role model.UserRole, actorUsername string) (*model.UserPublic, error) {
	username = strings.TrimSpace(username)
	if err := validateCredentials(username, password); err != nil {
		return nil, &UserActionError{Status: 400, Code: "USER_CREATE_INVALID", Message: err.Error()}
	}
	role, err := normalizeUserRole(role)
	if err != nil {
		return nil, err
	}

	existing, err := s.findUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, &UserActionError{Status: 409, Code: "USER_EXISTS", Message: "username already exists"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	record := model.UserRecord{
		ID:           fmt.Sprintf("usr_%d", time.Now().UnixNano()),
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.insertUser(record); err != nil {
		return nil, err
	}

	s.logAudit("auth.user_created", actorUsername, "created user "+username+" with role "+string(role))
	user := publicUser(record)
	return &user, nil
}

func (s *AuthService) UpdateUserRole(userID string, role model.UserRole, actor model.SessionClaims) (*model.UserPublic, error) {
	role, err := normalizeUserRole(role)
	if err != nil {
		return nil, err
	}

	record, err := s.findUserRecordByID(userID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, &UserActionError{Status: 404, Code: "USER_NOT_FOUND", Message: "user not found"}
	}

	if record.Role == model.RoleAdmin && role != model.RoleAdmin {
		count, err := s.countUsersByRole(model.RoleAdmin)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, &UserActionError{Status: 409, Code: "LAST_ADMIN_REQUIRED", Message: "at least one administrator is required"}
		}
	}

	if record.Role == role {
		public := publicUser(*record)
		return &public, nil
	}

	if _, err := s.db.Exec(`UPDATE users SET role = ? WHERE id = ?`, role, userID); err != nil {
		return nil, fmt.Errorf("update user role: %w", err)
	}
	if err := s.revokeSessionsForUser(userID); err != nil {
		return nil, err
	}

	record.Role = role
	s.logAudit("auth.user_role_updated", actor.Username, "updated role for "+record.Username+" to "+string(role))
	public := publicUser(*record)
	return &public, nil
}

func (s *AuthService) ResetUserPassword(userID, password string, actorUsername string) (*model.UserPublic, error) {
	if err := security.ValidatePassword(password); err != nil {
		return nil, &UserActionError{Status: 400, Code: "PASSWORD_INVALID", Message: err.Error()}
	}

	record, err := s.findUserRecordByID(userID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, &UserActionError{Status: 404, Code: "USER_NOT_FOUND", Message: "user not found"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	if _, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, string(hash), userID); err != nil {
		return nil, fmt.Errorf("reset password: %w", err)
	}
	if err := s.revokeSessionsForUser(userID); err != nil {
		return nil, err
	}

	s.logAudit("auth.user_password_reset", actorUsername, "reset password for "+record.Username)
	public := publicUser(*record)
	return &public, nil
}

func (s *AuthService) DeleteUser(userID string, actor model.SessionClaims) error {
	record, err := s.findUserRecordByID(userID)
	if err != nil {
		return err
	}
	if record == nil {
		return &UserActionError{Status: 404, Code: "USER_NOT_FOUND", Message: "user not found"}
	}

	if record.Role == model.RoleAdmin {
		count, err := s.countUsersByRole(model.RoleAdmin)
		if err != nil {
			return err
		}
		if count <= 1 {
			return &UserActionError{Status: 409, Code: "LAST_ADMIN_REQUIRED", Message: "at least one administrator is required"}
		}
	}

	if err := s.revokeSessionsForUser(userID); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	s.logAudit("auth.user_deleted", actor.Username, "deleted user "+record.Username)
	return nil
}

func (s *AuthService) findUserByUsername(username string) (*model.UserRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, username, password_hash, role, created_at
		FROM users
		WHERE username = ?
	`, username)

	var user model.UserRecord
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	return &user, nil
}

func (s *AuthService) findUserRecordByID(id string) (*model.UserRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, username, password_hash, role, created_at
		FROM users
		WHERE id = ?
	`, id)

	var user model.UserRecord
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}

func (s *AuthService) insertUser(user model.UserRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO users (id, username, password_hash, role, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, user.ID, user.Username, user.PasswordHash, user.Role, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *AuthService) countUsersByRole(role model.UserRole) (int, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, role).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users by role: %w", err)
	}
	return count, nil
}

func (s *AuthService) revokeSessionsForUser(userID string) error {
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}
	return nil
}

func (s *AuthService) issueToken(user model.UserRecord) (string, error) {
	expiresAt := time.Now().Add(s.cookieTTL).UTC()
	claims := model.SessionClaims{
		SID:      "ses_" + uuid.NewString(),
		Sub:      user.ID,
		Username: user.Username,
		Role:     user.Role,
		Exp:      expiresAt.Unix(),
	}
	if err := s.insertSession(claims.SID, user.ID, expiresAt); err != nil {
		return "", err
	}
	return s.session.Issue(claims)
}

func publicUser(user model.UserRecord) model.UserPublic {
	return model.UserPublic{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

func validateCredentials(username, password string) error {
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if err := security.ValidatePassword(password); err != nil {
		return err
	}
	return nil
}

func normalizeUserRole(role model.UserRole) (model.UserRole, error) {
	normalized, ok := model.ParseUserRole(string(role))
	if !ok {
		return "", &UserActionError{Status: 400, Code: "ROLE_INVALID", Message: "role must be one of admin, operator, or viewer"}
	}
	return normalized, nil
}

func initAuthSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		username TEXT,
		detail TEXT,
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("initialize auth schema: %w", err)
	}
	return nil
}

const (
	loginRateWindow = 10 * time.Minute
	loginRateMax    = 5
)

func (s *AuthService) loginLimited(username, clientAddr string) (bool, time.Duration) {
	key := loginAttemptKey(username, clientAddr)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	attempts := pruneAttempts(s.attempts[key], now)
	s.attempts[key] = attempts
	if len(attempts) < loginRateMax {
		return false, 0
	}

	retryAfter := loginRateWindow - now.Sub(attempts[0])
	if retryAfter < 0 {
		retryAfter = 0
	}
	return true, retryAfter
}

func (s *AuthService) recordLoginFailure(username, clientAddr string) {
	key := loginAttemptKey(username, clientAddr)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	attempts := pruneAttempts(s.attempts[key], now)
	s.attempts[key] = append(attempts, now)
}

func (s *AuthService) clearLoginFailures(username, clientAddr string) {
	key := loginAttemptKey(username, clientAddr)

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.attempts, key)
}

func pruneAttempts(attempts []time.Time, now time.Time) []time.Time {
	filtered := attempts[:0]
	for _, attempt := range attempts {
		if now.Sub(attempt) < loginRateWindow {
			filtered = append(filtered, attempt)
		}
	}
	return filtered
}

func loginAttemptKey(username, clientAddr string) string {
	username = strings.ToLower(strings.TrimSpace(username))
	ip := clientIP(clientAddr)
	return username + "|" + ip
}

func clientIP(clientAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(clientAddr))
	if err == nil && host != "" {
		return host
	}
	if clientAddr == "" {
		return "unknown"
	}
	return strings.TrimSpace(clientAddr)
}

func (s *AuthService) logAudit(eventType, username, detail string) {
	username = strings.TrimSpace(username)
	detail = strings.TrimSpace(detail)

	_, err := s.db.Exec(`
		INSERT INTO audit_logs (event_type, username, detail, created_at)
		VALUES (?, ?, ?, ?)
	`, eventType, nullableString(username), nullableString(detail), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		// Audit logging must not break the primary request path.
		return
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (s *AuthService) insertSession(sessionID, userID string, expiresAt time.Time) error {
	if _, err := s.db.Exec(`
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, sessionID, userID, expiresAt.UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *AuthService) deleteExpiredSessions() error {
	if _, err := s.db.Exec(`
		DELETE FROM sessions
		WHERE expires_at <= ?
	`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}
