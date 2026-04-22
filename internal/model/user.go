package model

import "strings"

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
	RoleViewer   UserRole = "viewer"
)

type UserRecord struct {
	ID           string   `json:"id"`
	Username     string   `json:"username"`
	PasswordHash string   `json:"passwordHash"`
	Role         UserRole `json:"role"`
	CreatedAt    string   `json:"createdAt"`
}

type UserPublic struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Role      UserRole `json:"role"`
	CreatedAt string   `json:"createdAt"`
}

type SessionClaims struct {
	SID      string   `json:"sid"`
	Sub      string   `json:"sub"`
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
	Exp      int64    `json:"exp"`
}

func ParseUserRole(value string) (UserRole, bool) {
	switch UserRole(strings.ToLower(strings.TrimSpace(value))) {
	case RoleAdmin:
		return RoleAdmin, true
	case RoleOperator:
		return RoleOperator, true
	case RoleViewer:
		return RoleViewer, true
	default:
		return "", false
	}
}
