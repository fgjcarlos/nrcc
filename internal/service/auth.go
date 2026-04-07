package service

import (
	"database/sql"
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
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
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
