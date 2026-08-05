package data

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/ItakawaM/greenlight/internal/validator"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// emailUniqueConstraint is a unique constraint for users' emails in Postgres.
const emailUniqueConstraint = "users_email_key"

// User represents a single user record in the application.
type User struct {
	ID        int64     `json:"id" example:"1"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-01T00:00:00Z"`
	Name      string    `json:"name" example:"Jane Doe"`
	Email     string    `json:"email" example:"jane@example.com"`
	Password  password  `json:"-"`
	IsActive  bool      `json:"is_active" example:"true"`
	Version   int       `json:"-"`
}

// UserModelInterface defines the storage operations available for users.
type UserModelInterface interface {
	// Insert adds a new user record to the database; populates its
	// ID, CreatedAt and Version fields on success.
	// Returns ErrDuplicateEmail if a user with such an email already exists.
	Insert(ctx context.Context, user *User) error

	// GetByEmail retrieves a user record by its email.
	// Returns ErrRecordNotFound if no user with that email exists.
	GetByEmail(ctx context.Context, email string) (*User, error)

	// GetForToken retrieves a user that is associated with the provided token.
	// Returns ErrRecordNotFound if no such user exists or the token is expired.
	GetForToken(ctx context.Context, scope TokenScope, tokenPlaintext string) (*User, error)

	// Update persists changes to an existing user record, using the
	// user's Version field for optimistic concurrency control.
	// Returns ErrEditConflict if the record was modified concurrently since it was last read.
	// Returns ErrDuplicateEmail if the new email breaks the unique constraint.
	Update(ctx context.Context, user *User) error
}

// UserModel implements UserModelInterface.
type UserModel struct {
	db DBTX
}

// Insert implements UserModelInterface.
func (m *UserModel) Insert(ctx context.Context, user *User) error {
	statement :=
		`INSERT INTO users (name, email, password_hash, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version;`

	if err := m.db.QueryRow(ctx, statement, user.Name, user.Email, user.Password.hash, user.IsActive).
		Scan(&user.ID, &user.CreatedAt, &user.Version); err != nil {
		switch {
		case isUniqueViolationError(err, emailUniqueConstraint):
			return ErrDuplicateEmail

		default:
			return handleContextErrors(err)
		}
	}

	return nil
}

// GetByEmail implements UserModelInterface.
func (m *UserModel) GetByEmail(ctx context.Context, email string) (*User, error) {
	statement :=
		`SELECT id, created_at, name, email, password_hash, is_active, version
		FROM users
		WHERE email = $1;`

	var user User
	if err := m.db.QueryRow(ctx, statement, email).
		Scan(&user.ID, &user.CreatedAt, &user.Name, &user.Email, &user.Password.hash, &user.IsActive, &user.Version); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound

		default:
			return nil, handleContextErrors(err)
		}
	}

	return &user, nil
}

// GetForToken implements UserModelInterface.
func (m *UserModel) GetForToken(ctx context.Context, scope TokenScope, tokenPlaintext string) (*User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	statement :=
		`SELECT users.id, users.created_at, users.name, users.email, users.password_hash, users.is_active, users.version
		FROM users
		INNER JOIN tokens
		ON users.id = tokens.user_id
		WHERE tokens.hash = $1
		AND tokens.scope = $2
		AND tokens.expiry > $3;`

	var user User
	if err := m.db.QueryRow(ctx, statement, tokenHash[:], scope, time.Now()).
		Scan(&user.ID, &user.CreatedAt, &user.Name, &user.Email, &user.Password.hash, &user.IsActive, &user.Version); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound

		default:
			return nil, handleContextErrors(err)
		}
	}

	return &user, nil
}

// Update implements UserModelInterface.
func (m *UserModel) Update(ctx context.Context, user *User) error {
	statement :=
		`UPDATE users
		SET name = $1, email = $2, password_hash = $3, is_active = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version;`

	if err := m.db.QueryRow(ctx, statement, user.Name, user.Email, user.Password.hash, user.IsActive, user.ID, user.Version).
		Scan(&user.Version); err != nil {
		switch {
		case isUniqueViolationError(err, emailUniqueConstraint):
			return ErrDuplicateEmail

		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict

		default:
			return handleContextErrors(err)
		}
	}

	return nil
}

// password is a wrapper type around plaintext and hashed password values.
type password struct {
	plaintext *string
	hash      []byte
}

// Set sets the plaintext and hashed versions of the password.
func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil
}

// Matches checks whether the provided plaintext password matches
// with the hash.
func (p *password) Matches(plaintextPassword string) (bool, error) {
	if err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword)); err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil

		default:
			return false, err
		}
	}

	return true, nil
}

// ValidateEmail checks whether the provided email is not empty and is valid.
func ValidateEmail(v *validator.Validator, email string) {
	v.Check(validator.NotBlank(email), "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

// ValidatePasswordPlaintext checks whether the provided value is a valid password:
//  1. Between 8 and 72 bytes (bcrypt upper limit);
//  2. One uppercase character;
//  3. One lowercase character;
//  4. One digit;
//  5. One special character (#?!@$%^&*-).
//
// It allows for spaces in the password.
func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case strings.ContainsRune("#?!@$%^&*-", char):
			hasSpecial = true
		}
	}

	v.Check(hasUpper, "password", "must contain at least one uppercase character")
	v.Check(hasLower, "password", "must contain at least one lowercase character")
	v.Check(hasNumber, "password", "must contain at least one digit")
	v.Check(hasSpecial, "password", "must contain at least one special character (#?!@$%^&*-)")
}

// ValidateUser checks the following requirements:
//  1. Name is not blank and is at most 120 characters long;
//  2. Email is valid (See: ValidateEmail);
//  3. Password is valid (See: ValidatePasswordPlaintext);
//
// Panics if the provided user has no hashed password.
func ValidateUser(v *validator.Validator, user *User) {
	v.Check(validator.NotBlank(user.Name), "name", "must be provided")
	v.Check(validator.MaxChars(user.Name, 120), "name", "must not be more than 120 characters long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	// only occurs if a developer makes a mistake; safe guard
	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}
