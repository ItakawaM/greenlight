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
	Metrics    string     `json:"metrics" example:"unavailable"`
}

// @Summary      Show API health status
// @Description  Returns the current operating status of the API, along with the running environment and application version.
// @Tags         healthcheck
// @Produce      json
// @Success      200  {object} HealthCheckResponse  "System is healthy"
// @Failure      500  {object} ErrorResponse  		"Server error"
// @Failure      503  {object} HealthCheckResponse  "System is degraded"
// @Router       /healthcheck [get]
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthCheckResponse{
		Status: "available",
		SystemInfo: SystemInfo{
			Environment: app.config.Server.Environment,
			Version:     app.config.Server.Version,
		},
	}

	statusCode := http.StatusOK

	if app.config.Metrics.Enabled {
		if !app.metricsHealthy.Load() {
			resp.Metrics = "unavailable"
			resp.Status = "degraded"
			statusCode = http.StatusServiceUnavailable
		} else {
			resp.Metrics = "available"
		}
	}

	if err := app.writeJSON(w, statusCode, resp, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
