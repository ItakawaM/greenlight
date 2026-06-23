package main

import (
	"errors"
	"net/http"
	"strconv"
)

func (app *application) readIDParam(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}
