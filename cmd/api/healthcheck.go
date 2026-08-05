package main

import (
	"net/http"
)

type SystemInfo struct {
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

// HealthCheckResponse represents the response type for healthcheck.
type HealthCheckResponse struct {
	Status     string     `json:"status"`
	SystemInfo SystemInfo `json:"system_info"`
}

// @Summary      Show API health status
// @Description  Returns the current operating status of the API, along with the running environment and application version.
// @Tags         healthcheck
// @Produce      json
// @Success      200  "System is healthy"
// @Failure      500  "Server error"
// @Router       /healthcheck [get]
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthCheckResponse{
		Status: "available",
		SystemInfo: SystemInfo{
			Environment: app.config.environment,
			Version:     version,
		},
	}

	if err := app.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
