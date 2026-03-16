package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParsePagination_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/items", nil)

	p := ParsePagination(c)
	if p.Limit != DefaultPageLimit {
		t.Errorf("expected default limit %d, got %d", DefaultPageLimit, p.Limit)
	}
	if p.Offset != 0 {
		t.Errorf("expected offset 0, got %d", p.Offset)
	}
}

func TestParsePagination_CustomValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/items?limit=25&offset=10", nil)

	p := ParsePagination(c)
	if p.Limit != 25 {
		t.Errorf("expected limit 25, got %d", p.Limit)
	}
	if p.Offset != 10 {
		t.Errorf("expected offset 10, got %d", p.Offset)
	}
}

func TestParsePagination_ClampMax(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/items?limit=500", nil)

	p := ParsePagination(c)
	if p.Limit != MaxPageLimit {
		t.Errorf("expected clamped limit %d, got %d", MaxPageLimit, p.Limit)
	}
}

func TestParsePagination_InvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/items?limit=abc&offset=-5", nil)

	p := ParsePagination(c)
	if p.Limit != DefaultPageLimit {
		t.Errorf("expected default limit for invalid input, got %d", p.Limit)
	}
	if p.Offset != 0 {
		t.Errorf("expected offset 0 for negative, got %d", p.Offset)
	}
}

func TestNewPaginationMeta(t *testing.T) {
	tests := []struct {
		name    string
		params  PaginationParams
		total   int
		hasMore bool
	}{
		{"middle page", PaginationParams{Limit: 10, Offset: 10}, 50, true},
		{"last page", PaginationParams{Limit: 10, Offset: 40}, 50, false},
		{"exact end", PaginationParams{Limit: 10, Offset: 40}, 50, false},
		{"beyond end", PaginationParams{Limit: 10, Offset: 60}, 50, false},
		{"empty", PaginationParams{Limit: 10, Offset: 0}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewPaginationMeta(tt.params, tt.total)
			if m.HasMore != tt.hasMore {
				t.Errorf("HasMore = %v, want %v", m.HasMore, tt.hasMore)
			}
			if m.Total != tt.total {
				t.Errorf("Total = %d, want %d", m.Total, tt.total)
			}
		})
	}
}

func TestRespondWithList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/items", nil)

	items := []string{"a", "b", "c"}
	meta := PaginationMeta{Limit: 10, Offset: 0, Total: 3, HasMore: false}
	RespondWithList(c, "items", items, meta)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := body["items"]; !ok {
		t.Error("response missing 'items' key")
	}
	if _, ok := body["pagination"]; !ok {
		t.Error("response missing 'pagination' key")
	}
}
