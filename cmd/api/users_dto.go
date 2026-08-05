package main

import "github.com/ItakawaM/greenlight/internal/data"

// CreateUserRequest represents a request object for user creation.
type CreateUserRequest struct {
	Name     string `json:"name" example:"Jane Doe"`
	Email    string `json:"email" example:"jane@example.com"`
	Password string `json:"password" example:"P@ssw0rd!"`
}

// ActivateUserRequest represents a request object for user activation.
type ActivateUserRequest struct {
	TokenPlaintext string `json:"token" example:"QWERTYUIOPASDFGHJKLZXCVBNM"`
}

// UserResponse represents a wrapped response containing a single user object.
type UserResponse struct {
	User *data.User `json:"user"`
}
