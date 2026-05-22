package handlers

import (
	"net/http"
	"strconv"
)

const DefaultPageSize = 20
const MaxPageSize = 100

// ParsePagination extracts offset and limit from query params
func ParsePagination(r *http.Request) (offset, limit int) {
	offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	return offset, limit
}

// PaginateSlice returns a sub-slice with pagination applied
func PaginateSlice[T any](items []T, offset, limit int) ([]T, int) {
	total := len(items)
	if offset >= total {
		return []T{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total
}
