package main

import (
	"errors"
	"net/http"

	"github.com/ItakawaM/greenlight/internal/data"
)

// logError writes an error to application's standard error logger.
//
// TODO: Log detailed request data as well.
func (app *application) logError(r *http.Request, err error) {
	app.logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.RequestURI(),
	})
}

// errorResponse sends a wrapped JSON error message with a provided status code.
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := envelope{"error": message}

	if err := app.writeJSON(w, status, env, nil); err != nil {
		app.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// serverErrorResponse logs the provided error and sends a default 500 JSON response.
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)

	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

// notFoundResponse sends a default 404 JSON response.
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}

// badRequestResponse sends a 400 JSON response with the provided error text.
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
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

// handleModelError is a model independent error handler that should be used after
// any model method. Checks for DB timeout, client cancellation, ordinary model errors and server errors.
func (app *application) handleModelError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, data.ErrTimeout):
		app.errorResponse(w, r, http.StatusGatewayTimeout, "the server took too long to process your request")
	case errors.Is(err, data.ErrCanceled):
		app.logger.PrintInfo("request canceled by client", map[string]string{
			"request_method": r.Method,
			"request_url":    r.URL.RequestURI(),
		})
	case errors.Is(err, data.ErrRecordNotFound):
		app.notFoundResponse(w, r)
	case errors.Is(err, data.ErrEditConflict):
		app.editConflictResponse(w, r)
	default:
		app.serverErrorResponse(w, r, err)
	}
}
