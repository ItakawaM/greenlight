package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ItakawaM/greenlight/internal/data"
	"github.com/ItakawaM/greenlight/internal/validator"
)

// @Summary      Register a new user
// @Description  Creates a new inactive user account and sends an activation email containing an activation token.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        input  body  object{name=string,email=string,password=string}  true  "User registration details"
// @Success      202  "User registered successfully. Activation email sent."
// @Failure      400  "Malformed request body"
// @Failure      422  "Validation failed or email already exists"
// @Failure      500  "Server encountered a problem"
// @Router       /users [post]
func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Name:     input.Name,
		Email:    input.Email,
		IsActive: false,
	}

	if err := user.Password.Set(input.Password); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// we run user creation and token assignment in a transaction
	// to avoid corrupted states if one of the operations fail
	// and allow for rollback
	var token *data.Token
	if err := app.models.WithTx(ctx, func(m *data.Models) error {
		var err error
		if err := m.Users.Insert(ctx, user); err != nil {
			return err
		}

		token, err = m.Tokens.New(ctx, user.ID, 72*time.Hour, data.ScopeActivation)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)

		default:
			app.handleModelError(w, r, err)
		}
		return
	}

	app.background(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := app.mailer.Send(ctx, "user_welcome.tmpl.html", map[string]string{
			"name": user.Name, "activationToken": token.Plaintext,
		}, user.Email); err != nil {
			app.logger.ErrorContext(ctx, "failed email delivery",
				slog.String("error", err.Error()))
		}
	})

	if err := app.writeJSON(w, http.StatusAccepted, envelope{"user": user}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// @Summary      Activate a user account
// @Description  Activates a user account using a valid activation token. The activation token is deleted after successful activation.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        input  body  object{token=string}  true  "Activation token"
// @Success      200  "User activated successfully"
// @Failure      400  "Malformed request body"
// @Failure      422  "Validation failed or activation token is invalid/expired"
// @Failure      500  "Server encountered a problem"
// @Router       /users/activate [put]
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// we run the following operations in a transaction
	// to avoid corrupted states if one of the operations fail
	// and allow for rollback
	var user *data.User
	if err := app.models.WithTx(ctx, func(m *data.Models) error {
		var err error
		user, err = m.Users.GetForToken(ctx, data.ScopeActivation, input.TokenPlaintext)
		if err != nil {
			return err
		}

		user.IsActive = true
		if err := m.Users.Update(ctx, user); err != nil {
			return err
		}

		if err := m.Tokens.DeleteAllForUser(ctx, user.ID, data.ScopeActivation); err != nil {
			return err
		}

		return nil
	}); err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			app.failedValidationResponse(w, r, v.Errors)

		default:
			app.handleModelError(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
