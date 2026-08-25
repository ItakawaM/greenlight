package main

import "net/http"

// healthcheckHandler is a simple dummy healtcheck that always responds with OK.
func (e *exporter) healthcheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
