package handler

import (
	"net/http/httptest"
	"testing"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		query   string
		want    Pagination
		badKeys []string
	}{
		{"", Pagination{1, 20}, nil},
		{"page=3&per_page=100", Pagination{3, 100}, nil},
		{"per_page=101", Pagination{1, 20}, []string{"per_page"}},
		{"page=0", Pagination{1, 20}, []string{"page"}},
		{"per_page=0", Pagination{1, 20}, []string{"per_page"}},
		{"page=-1&per_page=-5", Pagination{1, 20}, []string{"page", "per_page"}},
		{"page=abc&per_page=1.5", Pagination{1, 20}, []string{"page", "per_page"}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got, errs := ParsePagination(httptest.NewRequest("GET", "/?"+tt.query, nil))
			if got != tt.want {
				t.Fatalf("pagination = %+v, want %+v", got, tt.want)
			}
			if len(errs) != len(tt.badKeys) {
				t.Fatalf("errs = %v, want keys %v", errs, tt.badKeys)
			}
			for _, k := range tt.badKeys {
				if errs[k] == "" {
					t.Fatalf("missing field error %q in %v", k, errs)
				}
			}
		})
	}
}

func TestPaginationOffset(t *testing.T) {
	if got := (Pagination{Page: 3, PerPage: 20}).Offset(); got != 40 {
		t.Fatalf("offset = %d, want 40", got)
	}
}
