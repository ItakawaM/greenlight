package main

import "github.com/ItakawaM/greenlight/internal/data"

// CreateUserRequest represents a request object for user creation.
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ActivateUserRequest represents a request object for user activation.
type ActivateUserRequest struct {
	TokenPlaintext string `json:"token"`
}

// UserResponse represents a wrapped response containing a single user object.
type UserResponse struct {
	User *data.User `json:"user"`
}
