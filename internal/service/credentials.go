package service

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/google/uuid"
)

var (
	ErrCredentialsChanged = errors.New("credentials changed")
	ErrReplacementSession = errors.New("could not stage replacement session")
	ErrInvalidRefresh     = errors.New("invalid refresh session")
)

// The zero version preserves tokens and sessions issued before versioning.
func credentialVersion(user *model.CCUser) string {
	if user.PasswordVersion == "" && user.MfaVersion == "" {
		return ""
	}
	return user.PasswordVersion + ":" + user.MfaVersion
}

func (s *AuthService) mfaVersions() (map[string]string, error) {
	if s.mfaStore == nil {
		return nil, nil
	}
	state, err := s.mfaStore.Read()
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return state.CredentialVersions, err
}

func (s *AuthService) withMfaVersion(user model.CCUser) (*model.CCUser, error) {
	versions, err := s.mfaVersions()
	if err != nil {
		return nil, err
	}
	user.MfaVersion = versions[user.ID]
	return &user, nil
}

// CredentialChangeSession contains replacement credentials for self-service only.
type CredentialChangeSession struct {
	AccessToken  string
	RefreshToken string
}

// stageCredentialSession writes a session for a never-used version first.
// It is unusable until the credential update commits; on failure it stays
// unusable and expires normally. Existing sessions are untouched until commit.
func (s *AuthService) stageCredentialSession(user *model.CCUser, self bool) (CredentialChangeSession, error) {
	if !self {
		return CredentialChangeSession{}, nil
	}
	access, err := s.GenerateToken(user)
	if err != nil {
		return CredentialChangeSession{}, fmt.Errorf("%w: %v", ErrReplacementSession, err)
	}
	refresh, err := s.CreateRefreshSessionForUser(user)
	if err != nil {
		return CredentialChangeSession{}, fmt.Errorf("%w: %v", ErrReplacementSession, err)
	}
	return CredentialChangeSession{AccessToken: access, RefreshToken: refresh}, nil
}

// ChangePassword commits the hash and revocation version in the same users
// write. For self-service, expected is the credential snapshot reauthenticated
// by the caller before hashing the new password.
func (s *AuthService) ChangePassword(userID, hash string, expected *model.CCUser) (string, CredentialChangeSession, error) {
	s.credentialMu.Lock()
	defer s.credentialMu.Unlock()

	user := s.GetUserByID(userID)
	if user == nil {
		return "", CredentialChangeSession{}, ErrUserNotFound
	}
	if expected != nil && credentialVersion(user) != credentialVersion(expected) {
		return user.Username, CredentialChangeSession{}, ErrCredentialsChanged
	}
	user.PasswordVersion = uuid.NewString()
	replacement, err := s.stageCredentialSession(user, expected != nil)
	if err != nil {
		return user.Username, CredentialChangeSession{}, err
	}
	err = s.UpdateUser(userID, func(current *model.CCUser) error {
		current.PasswordHash = hash
		current.PasswordVersion = user.PasswordVersion
		current.UpdatedAt = model.NowISO8601()
		return nil
	})
	return user.Username, replacement, err
}
