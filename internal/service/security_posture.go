package service

import (
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// SecurityPosture is the canonical server-derived security snapshot.
type SecurityPosture struct {
	EncryptionKeyConfigured bool               `json:"encryptionKeyConfigured"`
	BackupDownloadAdminOnly bool               `json:"backupDownloadAdminOnly"`
	ActiveRefreshSessions   int                `json:"activeRefreshSessions"`
	MFA                     SecurityPostureMFA `json:"mfa"`
}

// SecurityPostureMFA reports confirmed enrollment among current admins.
type SecurityPostureMFA struct {
	EnrolledAdmins int `json:"enrolledAdmins"`
	TotalAdmins    int `json:"totalAdmins"`
}

type securityPostureAuthReader interface {
	CountActiveRefreshSessions(time.Time) (int, error)
	GetAllUsers() ([]model.CCUser, error)
}

type securityPostureMFAReader interface {
	CountEnrolled([]string) (int, error)
}

// SecurityPostureService collects read-only posture data from auth and MFA stores.
type SecurityPostureService struct {
	encryptionKeyConfigured bool
	backupDownloadAdminOnly bool
	auth                    securityPostureAuthReader
	mfa                     securityPostureMFAReader
}

// NewSecurityPostureService creates a posture snapshot service from startup policy.
func NewSecurityPostureService(encryptionKeyConfigured, backupDownloadAdminOnly bool, auth securityPostureAuthReader, mfa securityPostureMFAReader) *SecurityPostureService {
	return &SecurityPostureService{
		encryptionKeyConfigured: encryptionKeyConfigured,
		backupDownloadAdminOnly: backupDownloadAdminOnly,
		auth:                    auth,
		mfa:                     mfa,
	}
}

// Snapshot returns current canonical posture values.
func (s *SecurityPostureService) Snapshot(now time.Time) (SecurityPosture, error) {
	activeSessions, err := s.auth.CountActiveRefreshSessions(now)
	if err != nil {
		return SecurityPosture{}, fmt.Errorf("count active refresh sessions: %w", err)
	}
	users, err := s.auth.GetAllUsers()
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return SecurityPosture{}, fmt.Errorf("read users: %w", err)
		}
		users = nil
	}
	adminIDs := make([]string, 0, len(users))
	for _, user := range users {
		if user.Role == model.RoleAdmin {
			adminIDs = append(adminIDs, user.ID)
		}
	}
	enrolledAdmins, err := s.mfa.CountEnrolled(adminIDs)
	if err != nil {
		return SecurityPosture{}, fmt.Errorf("count enrolled admins: %w", err)
	}

	return SecurityPosture{
		EncryptionKeyConfigured: s.encryptionKeyConfigured,
		BackupDownloadAdminOnly: s.backupDownloadAdminOnly,
		ActiveRefreshSessions:   activeSessions,
		MFA: SecurityPostureMFA{
			EnrolledAdmins: enrolledAdmins,
			TotalAdmins:    len(adminIDs),
		},
	}, nil
}
