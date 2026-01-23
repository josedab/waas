package marketplace

import (
	"testing"
)

func TestGetBuiltinCatalog(t *testing.T) {
	catalog := GetBuiltinCatalog()
	if len(catalog) == 0 {
		t.Fatal("expected non-empty catalog")
	}

	totalTemplates := 0
	for _, cat := range catalog {
		if cat.ID == "" || cat.Name == "" {
			t.Errorf("category missing ID or Name: %+v", cat)
		}
		if len(cat.Templates) == 0 {
			t.Errorf("category '%s' has no templates", cat.Name)
		}
		totalTemplates += len(cat.Templates)
	}

	if totalTemplates < 15 {
		t.Errorf("expected at least 15 templates, got %d", totalTemplates)
	}
}

func TestSearchCatalog(t *testing.T) {
	tests := []struct {
		query    string
		minCount int
	}{
		{"stripe", 1},
		{"payment", 4},
		{"github", 1},
		{"shopify", 1},
		{"nonexistent-provider-xyz", 0},
		{"", 15}, // empty query returns all
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results := SearchCatalog(tt.query)
			if len(results) < tt.minCount {
				t.Errorf("SearchCatalog(%q) returned %d results, want at least %d", tt.query, len(results), tt.minCount)
			}
		})
	}
}

func TestGetTemplateBySlug(t *testing.T) {
	tmpl := GetTemplateBySlug("stripe")
	if tmpl == nil {
		t.Fatal("expected to find stripe template")
	}
	if tmpl.Name != "Stripe" {
		t.Errorf("expected name 'Stripe', got '%s'", tmpl.Name)
	}
	if len(tmpl.EventTypes) == 0 {
		t.Error("expected stripe template to have event types")
	}

	if GetTemplateBySlug("nonexistent") != nil {
		t.Error("expected nil for nonexistent slug")
	}
}

func TestGetTemplatesByCategory(t *testing.T) {
	payments := GetTemplatesByCategory("payments")
	if len(payments) < 3 {
		t.Errorf("expected at least 3 payment templates, got %d", len(payments))
	}

	none := GetTemplatesByCategory("nonexistent")
	if none != nil {
		t.Errorf("expected nil for nonexistent category, got %d templates", len(none))
	}
}

func TestCatalogTemplatesAreVerified(t *testing.T) {
	for _, cat := range GetBuiltinCatalog() {
		for _, tmpl := range cat.Templates {
			if !tmpl.Verified {
				t.Errorf("template '%s' should be verified", tmpl.Name)
			}
			if tmpl.ID == "" {
				t.Errorf("template in category '%s' missing ID", cat.Name)
			}
		}
	}
}
