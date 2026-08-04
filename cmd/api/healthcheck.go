package main

import (
	"net/http"
)

// @Summary      Show API health status
// @Description  Returns the current operating status of the API, along with the running environment and application version.
// @Tags         healthcheck
// @Produce      json
// @Success      200  "System is healthy"
// @Failure      500  "Server error"
// @Router       /healthcheck [get]
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
