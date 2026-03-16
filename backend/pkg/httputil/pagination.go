package httputil

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPageLimit = 50
	MaxPageLimit     = 100
)

// PaginationParams holds parsed pagination query parameters.
type PaginationParams struct {
	Limit  int
	Offset int
}

// ParsePagination extracts and validates "limit" and "offset" query parameters
// from a Gin request. Invalid or out-of-range values are silently clamped.
func ParsePagination(c *gin.Context) PaginationParams {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(DefaultPageLimit)))
	if err != nil || limit <= 0 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	return PaginationParams{Limit: limit, Offset: offset}
}

// PaginationMeta is the standard pagination metadata included in list
// responses across all API endpoints.
type PaginationMeta struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	HasMore bool `json:"has_more"`
}

// NewPaginationMeta builds a PaginationMeta from the request params and total
// count returned by the data layer.
func NewPaginationMeta(p PaginationParams, total int) PaginationMeta {
	return PaginationMeta{
		Limit:   p.Limit,
		Offset:  p.Offset,
		Total:   total,
		HasMore: p.Offset+p.Limit < total,
	}
}

// RespondWithList is a convenience helper that sends a JSON list response with
// consistent pagination metadata. The `key` argument is the top-level JSON
// field name for the items (e.g. "tenants", "endpoints").
func RespondWithList(c *gin.Context, key string, items interface{}, meta PaginationMeta) {
	c.JSON(http.StatusOK, gin.H{
		key:          items,
		"pagination": meta,
	})
}
