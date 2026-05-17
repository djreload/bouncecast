package models

import (
	"time"

	"github.com/owncast/owncast/utils"
)

type User struct {
	CreatedAt       time.Time  `json:"createdAt"`
	DisabledAt      *time.Time `json:"disabledAt,omitempty"`
	NameChangedAt   *time.Time `json:"nameChangedAt,omitempty"`
	AuthenticatedAt *time.Time `json:"-"`
	RegisteredAt    *time.Time `json:"registeredAt,omitempty"`
	LastLoginAt     *time.Time `json:"lastLoginAt,omitempty"`
	ID              string     `json:"id"`
	DisplayName     string     `json:"displayName"`
	Email           string     `json:"email,omitempty"`
	ProfileImageURL string     `json:"profileImageUrl,omitempty"`
	PreviousNames   []string   `json:"previousNames"`
	Scopes          []string   `json:"scopes,omitempty"`
	DisplayColor    int        `json:"displayColor"`
	IsBot           bool       `json:"isBot"`
	Authenticated   bool       `json:"authenticated"`
}

// IsEnabled will return if this single user is enabled.
func (u *User) IsEnabled() bool {
	return u.DisabledAt == nil
}

// HasScope returns if the user has a specific access scope.
func (u *User) HasScope(scope string) bool {
	_, hasScope := utils.FindInSlice(u.Scopes, scope)
	return hasScope
}

// IsModerator will return if the user has moderation privileges.
func (u *User) IsModerator() bool {
	return u.HasScope(ModeratorScopeKey)
}

// IsOwner will return if the user has owner privileges.
func (u *User) IsOwner() bool {
	return u.HasScope(BounceCastOwnerScopeKey)
}

// IsAdmin will return if the user has BounceCast admin privileges.
func (u *User) IsAdmin() bool {
	return u.HasScope(BounceCastAdminScopeKey)
}

// IsDJ will return if the user has DJ privileges.
func (u *User) IsDJ() bool {
	return u.HasScope(BounceCastDJScopeKey)
}
