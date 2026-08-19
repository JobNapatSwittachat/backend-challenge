// Package domain contains the core business entities of the application.
// It has no dependencies on frameworks, drivers, or other layers.
package domain

import "time"

// User is the core business entity representing an application user.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserUpdate carries the mutable fields of a user. Nil means "no change".
type UserUpdate struct {
	Name  *string
	Email *string
}
