package versioning

import (
	"testing"
)

func TestSemanticVersion_String(t *testing.T) {
	v := SemanticVersion{Major: 1, Minor: 2, Patch: 3}
	if s := v.String(); s != "1.2.3" {
		t.Errorf("String() = %q, want %q", s, "1.2.3")
	}
}

func TestSemanticVersion_Compare(t *testing.T) {
	tests := []struct {
		a, b SemanticVersion
		want int
	}{
		{SemanticVersion{Major: 1, Minor: 0, Patch: 0}, SemanticVersion{Major: 1, Minor: 0, Patch: 0}, 0},
		{SemanticVersion{Major: 2, Minor: 0, Patch: 0}, SemanticVersion{Major: 1, Minor: 0, Patch: 0}, 1},
		{SemanticVersion{Major: 1, Minor: 0, Patch: 0}, SemanticVersion{Major: 2, Minor: 0, Patch: 0}, -1},
		{SemanticVersion{Major: 1, Minor: 2, Patch: 0}, SemanticVersion{Major: 1, Minor: 1, Patch: 0}, 1},
		{SemanticVersion{Major: 1, Minor: 1, Patch: 0}, SemanticVersion{Major: 1, Minor: 2, Patch: 0}, -1},
		{SemanticVersion{Major: 1, Minor: 1, Patch: 2}, SemanticVersion{Major: 1, Minor: 1, Patch: 1}, 1},
		{SemanticVersion{Major: 1, Minor: 1, Patch: 1}, SemanticVersion{Major: 1, Minor: 1, Patch: 2}, -1},
	}

	for _, tt := range tests {
		t.Run(tt.a.String()+" vs "+tt.b.String(), func(t *testing.T) {
			if got := tt.a.Compare(tt.b); got != tt.want {
				t.Errorf("Compare() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSemanticVersion_Bump(t *testing.T) {
	tests := []struct {
		input    SemanticVersion
		bumpType string
		want     SemanticVersion
	}{
		{SemanticVersion{Major: 1, Minor: 2, Patch: 3}, "major", SemanticVersion{Major: 2, Minor: 0, Patch: 0}},
		{SemanticVersion{Major: 1, Minor: 2, Patch: 3}, "minor", SemanticVersion{Major: 1, Minor: 3, Patch: 0}},
		{SemanticVersion{Major: 1, Minor: 2, Patch: 3}, "patch", SemanticVersion{Major: 1, Minor: 2, Patch: 4}},
	}

	for _, tt := range tests {
		t.Run(tt.bumpType, func(t *testing.T) {
			got := tt.input.Bump(tt.bumpType)
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
				t.Errorf("Bump(%q) = %v, want %v", tt.bumpType, got, tt.want)
			}
		})
	}
}

func TestVersion_Status(t *testing.T) {
	tests := []struct {
		status VersionStatus
		want   bool
	}{
		{VersionDraft, false},
		{VersionActive, true},
		{VersionDeprecated, true},
		{VersionSunset, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			v := &Version{Status: tt.status}
			// Check status directly
			isActive := v.Status == VersionActive || v.Status == VersionDeprecated
			if isActive != tt.want {
				t.Errorf("expected active=%v for status %s", tt.want, tt.status)
			}
		})
	}
}

func TestVersionStatus_Constants(t *testing.T) {
	statuses := []VersionStatus{VersionDraft, VersionActive, VersionDeprecated, VersionSunset}
	for _, s := range statuses {
		if s == "" {
			t.Error("VersionStatus should not be empty")
		}
	}
}
