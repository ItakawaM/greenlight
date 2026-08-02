package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/ItakawaM/greenlight/internal/validator"
)

// TokenScope is a representation of a type of a token.
type TokenScope string

const (
	// ScopeActivation is used during account activation.
	ScopeActivation TokenScope = "activation"
)

// Token represents a single token record in the application.
type Token struct {
	Plaintext string
	Hash      []byte
	UserID    int64
	Expiry    time.Time
	Scope     TokenScope
}

// TokenModelInterface defines the storage operations available for tokens.
type TokenModelInterface interface {
	// New generates a new token for the specified user with the provided ttl and scope.
	// Adds the token record to the database.
	New(ctx context.Context, userID int64, ttl time.Duration, scope TokenScope) (*Token, error)

	// Insert adds a new token record to the database.
	Insert(ctx context.Context, token *Token) error

	// DeleteAllForUser removes all the tokens for the specified user with the provided scope.
	DeleteAllForUser(ctx context.Context, userID int64, scope TokenScope) error
}

// TokenModel implements TokenModelInterface.
type TokenModel struct {
	db DBTX
}

// New implements TokenModelInterface.
func (m *TokenModel) New(ctx context.Context, userID int64, ttl time.Duration, scope TokenScope) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(ctx, token)
	return token, err // we don't need to wrap the error in handleContextErrors, because Insert takes care of that
}

// Insert implements TokenModelInterface.
func (m *TokenModel) Insert(ctx context.Context, token *Token) error {
	statement :=
		`INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4);`

	_, err := m.db.Exec(ctx, statement, token.Hash, token.UserID, token.Expiry, token.Scope)
	return handleContextErrors(err)
}

// DeleteAllForUser implements TokenModelInterface.
func (m *TokenModel) DeleteAllForUser(ctx context.Context, userID int64, scope TokenScope) error {
	statement :=
		`DELETE FROM tokens
	WHERE scope = $1 AND user_id = $2;`

	_, err := m.db.Exec(ctx, statement, scope, userID)
	return handleContextErrors(err)
}

// generateToken creates a cryptographically secure base32 encoded token
// for the specified user with the provided ttl and scope.
// Creates a sha256 hashsum of the plaintext token.
func generateToken(userID int64, ttl time.Duration, scope TokenScope) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, err
	}

	// we encode randomly generated 16 bytes using base32 and then store its hash
	// the base32 string will be the token our users use
	// while we just match hashes
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// ValidateTokenPlaintext checks whether the provided token is not empty and consists of 26 bytes.
func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(validator.NotBlank(tokenPlaintext), "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}
