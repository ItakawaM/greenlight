package main

import (
	"net/http"
)

type SystemInfo struct {
	Environment string `json:"environment" example:"production"`
	Version     string `json:"version" example:"v1.0.0"`
}

// HealthCheckResponse represents the response type for healthcheck.
type HealthCheckResponse struct {
	Status     string     `json:"status" example:"available"`
	SystemInfo SystemInfo `json:"system_info"`
}

// @Summary      Show API health status
// @Description  Returns the current operating status of the API, along with the running environment and application version.
// @Tags         healthcheck
// @Produce      json
// @Success      200  {object} HealthCheckResponse  "System is healthy"
// @Failure      500  {object} ErrorResponse  "Server error"
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
