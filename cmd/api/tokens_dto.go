package main

import "github.com/ItakawaM/greenlight/internal/data"

// CreateAuthenticationTokenRequest represents a request object for authentication.
type CreateAuthenticationTokenRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateAuthenticationTokenRequest represents a response object for authentication.
type CreateAuthenticationTokenResponse struct {
	AuthenticationToken *data.Token `json:"authentication_token"`
}
