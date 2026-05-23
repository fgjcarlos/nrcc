package model

// UserRole represents the user's role
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleViewer UserRole = "viewer"
)

// CCUser represents a control center user
type CCUser struct {
	ID           string   `json:"id"`
	Username     string   `json:"username"`
	PasswordHash string   `json:"passwordHash"`
	Role         UserRole `json:"role"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

// CCUserPublic is the public representation of a user (no password hash)
type CCUserPublic struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Role      UserRole `json:"role"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

// CCUsers is the root structure for cc-users.json
type CCUsers struct {
	Users []CCUser `json:"users"`
}

// Claims represents JWT claims
type Claims struct {
	UserID    string   `json:"userId"`
	Username  string   `json:"username"`
	Role      UserRole `json:"role"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
}

// RefreshSession represents a server-side refresh session.
type RefreshSession struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	ExpiresAt int64  `json:"expiresAt"`
	Revoked   bool   `json:"revoked"`
	CreatedAt int64  `json:"createdAt"`
}

// RefreshSessions is the root structure for refresh_sessions.json.
type RefreshSessions struct {
	Sessions []RefreshSession `json:"sessions"`
}
