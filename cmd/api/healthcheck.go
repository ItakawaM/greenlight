package main

import (
	"net/http"
)

// healthcheckHandler responds with system data about API.
//
// Route: GET /v1/healthcheck
//
// Responses:
//   - 200 (OK): Data was successfully sent.
//   - 500 (Internal Server Error): Server encountered a problem.
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	data := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.environment,
			"version":     version,
		},
	}

	if err := app.writeJSON(w, http.StatusOK, data, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
