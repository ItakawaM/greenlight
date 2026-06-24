package main

import (
	"net/http"
)

// logError writes an error to application's standard error logger.
//
// TODO: Log detailed request data as well.
func (app *application) logError(r *http.Request, err error) {
	app.errorLogger.Print(err)
}

// errorResponse sends a wrapped JSON error message with a provided status code.
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	data := envelope{"error": message}

	if err := app.writeJSON(w, status, data, nil); err != nil {
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

// failedValidationResponse sends a 422 JSON response with the provided errors object.
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}
