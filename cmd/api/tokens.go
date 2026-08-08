package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ItakawaM/greenlight/internal/data"
	"github.com/ItakawaM/greenlight/internal/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateAuthenticationTokenRequest
	if err := app.readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, req.Email)
	data.ValidatePasswordPlaintext(v, req.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var token *data.Token
	if err := app.models.WithTx(ctx, func(m *data.Models) error {
		var err error
		user, err := m.Users.GetByEmail(ctx, req.Email)
		if err != nil {
			return err
		}

		ok, err := user.Password.Matches(req.Password)
		if err != nil {
			return err
		}

		if !ok {
			return data.ErrRecordNotFound
		}

		token, err = m.Tokens.New(ctx, user.ID, 24*time.Hour, data.ScopeAuthentication)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)

		default:
			app.handleContextErrors(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusCreated, CreateAuthenticationTokenResponse{AuthenticationToken: token}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
