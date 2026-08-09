package main

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ItakawaM/greenlight/internal/data"
)

// ErrorResponse represents a universal response type for errors.
type ErrorResponse struct {
	Error any `json:"error"`
}

// logError writes an error to application's standard error logger
// and records metadata about the request.
func (app *application) logRequestError(r *http.Request, err error, attrs ...slog.Attr) {
	all := make([]slog.Attr, 0, len(attrs)+2)
	all = append(all,
		slog.String("request_method", r.Method),
		slog.String("request_url", r.URL.RequestURI()),
	)
	all = append(all, attrs...)

	app.logger.LogAttrs(r.Context(), slog.LevelError, err.Error(), all...)
}

// errorResponse sends a wrapped JSON error message with a provided status code.
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	resp := ErrorResponse{
		Error: message,
	}

	if err := app.writeJSON(w, status, resp, nil); err != nil {
		app.logRequestError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// badRequestResponse sends a 400 JSON response with the provided error text.
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

// invalidCredentialsResponse sends a 401 invalid credentials JSON response.
func (app *application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// invalidAuthenticationTokenResponse sends a 401 invalid authenticaton token JSON response.
func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// authenticationRequiredResponse sends a 401 missing authentication JSON response.
func (app *application) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "you must be authenticated to access this resource"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// notPermittedResponse sends a 403 missing permissions JSON response.
func (app *application) notPermittedResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account doesn't have the necessary permissions to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}

// inactiveAccountResponse sends a 403 inactive user JSON response.
func (app *application) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}

// notFoundResponse sends a 404 JSON response.
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}

// editConflictResponse sends a 409 JSON response signifying a conflict (example: data race).
func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errorResponse(w, r, http.StatusConflict, message)
}

// failedValidationResponse sends a 422 JSON response with the provided errors object.
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

// rateLimitExceededResponse sends a 429 JSON response signifying that
// the client has hit rate limit.
func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}

// serverErrorResponse logs the provided error and sends a 500 JSON response.
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logRequestError(r, err)

	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

// handleContextErrors checks for DB timeout and client request cancellation.
func (app *application) handleContextErrors(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, data.ErrTimeout):
		app.errorResponse(w, r, http.StatusGatewayTimeout, "the server took too long to process your request")

	case errors.Is(err, data.ErrCanceled):
		app.logger.LogAttrs(r.Context(), slog.LevelInfo, "request canceled by client",
			slog.String("request_method", r.Method),
			slog.String("request_url", r.URL.RequestURI()),
		)

	default:
		app.serverErrorResponse(w, r, err)
	}
}
