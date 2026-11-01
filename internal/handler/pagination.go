package handler

import (
	"net/http"
	"strconv"
)

const (
	defaultPerPage = 20
	maxPerPage     = 100
)

// Pagination holds validated ?page and ?per_page values.
type Pagination struct {
	Page    int
	PerPage int
}

// Offset is the number of rows to skip for this page.
func (p Pagination) Offset() int { return (p.Page - 1) * p.PerPage }

// ParsePagination reads ?page and ?per_page, applying defaults when omitted.
// Invalid values are refused, not clamped: the returned map holds one Field
// error per bad param and is nil when all is valid.
func ParsePagination(r *http.Request) (Pagination, map[string]string) {
	q := r.URL.Query()
	p := Pagination{Page: 1, PerPage: defaultPerPage}
	var errs map[string]string
	fail := func(field, msg string) {
		if errs == nil {
			errs = map[string]string{}
		}
		errs[field] = msg
	}

	if v := q.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			fail("page", "must be an integer of 1 or more")
		} else {
			p.Page = n
		}
	}
	if v := q.Get("per_page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxPerPage {
			fail("per_page", "must be an integer from 1 to 100")
		} else {
			p.PerPage = n
		}
	}
	return p, errs
}
