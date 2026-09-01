package logging

import (
	"fmt"
	"log/slog"
)

// SlogProm is an adapter that implements promhttp.Logger.
type SlogProm struct {
	logger *slog.Logger
}

// NewSlogProm initializes a new prom logger.
func NewSlogProm(logger *slog.Logger) *SlogProm {
	return &SlogProm{
		logger: logger,
	}
}

// Println implements promhttp.Logger.
func (s *SlogProm) Println(v ...any) {
	s.logger.Error(fmt.Sprint(v...))
}
