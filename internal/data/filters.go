package data

import (
	"math"
	"slices"
	"strings"

	"github.com/ItakawaM/greenlight/internal/validator"
)

// Metadata represents pagination information that can be used by the client.
type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

// calculateMetadata creates a new Metadata object and calculates its LastPage.
func calculateMetadata(totalRecords int, page int, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}

	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}
}

// Filters represents filters that can be used for pagination and sorting.
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafeList []string
}

// limit is the amount of rows to be queried by the DB.
func (f *Filters) limit() int {
	return f.PageSize
}

// offset is the starting row to be queried from by the DB.
func (f *Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

// sortColumn returns a normalized column name to be sorted by.
// Panics if Sort is not in SortSafeList (validate first).
func (f *Filters) sortColumn() string {
	if slices.Contains(f.SortSafeList, f.Sort) {
		return strings.TrimPrefix(f.Sort, "-")
	}

	panic("unsafe sort parameter: " + f.Sort)
}

// sortDirection returns the direction to be sorted by denoted by the prefix.
func (f *Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

// ValidateMovie executes validation checks against a Filters instance, populating
// the provided Validator with any formatting or business-logic errors.
func ValidateFilters(v *validator.Validator, f *Filters) {
	v.Check(f.Page > 0, "page", "must be greather than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")

	v.Check(f.PageSize > 0, "page_size", "must be greather than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")

	v.Check(validator.PermittedValue(f.Sort, f.SortSafeList...), "sort", "invalid sort value")
}
