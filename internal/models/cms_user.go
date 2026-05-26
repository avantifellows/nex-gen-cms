package models

import "time"

type CmsUser struct {
	ID          int64      `json:"id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	FullName    *string    `json:"full_name,omitempty"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	InsertedAt  time.Time  `json:"inserted_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
