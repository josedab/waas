package database

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// StringArray is a []string that implements sql.Scanner and driver.Valuer
// for PostgreSQL array columns. This replaces pq.Array for string slices.
type StringArray []string

// Scan implements sql.Scanner for reading PostgreSQL text[] arrays.
func (a *StringArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}

	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("StringArray.Scan: unsupported type %T", src)
	}

	// Parse PostgreSQL array literal: {elem1,elem2,...}
	s = strings.TrimSpace(s)
	if s == "{}" || s == "" {
		*a = []string{}
		return nil
	}

	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return fmt.Errorf("StringArray.Scan: invalid array format: %q", s)
	}

	inner := s[1 : len(s)-1]
	if inner == "" {
		*a = []string{}
		return nil
	}

	*a = parseArrayElements(inner)
	return nil
}

// Value implements driver.Valuer for writing PostgreSQL text[] arrays.
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	if len(a) == 0 {
		return "{}", nil
	}

	escaped := make([]string, len(a))
	for i, elem := range a {
		escaped[i] = quoteArrayElement(elem)
	}
	return "{" + strings.Join(escaped, ",") + "}", nil
}

// UUIDArray is a []uuid.UUID that implements sql.Scanner and driver.Valuer
// for PostgreSQL uuid[] array columns.
type UUIDArray []uuid.UUID

// Scan implements sql.Scanner for reading PostgreSQL uuid[] arrays.
func (a *UUIDArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}

	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("UUIDArray.Scan: unsupported type %T", src)
	}

	s = strings.TrimSpace(s)
	if s == "{}" || s == "" {
		*a = []uuid.UUID{}
		return nil
	}

	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return fmt.Errorf("UUIDArray.Scan: invalid array format: %q", s)
	}

	elements := parseArrayElements(s[1 : len(s)-1])
	result := make([]uuid.UUID, 0, len(elements))
	for _, elem := range elements {
		id, err := uuid.Parse(elem)
		if err != nil {
			return fmt.Errorf("UUIDArray.Scan: invalid UUID %q: %w", elem, err)
		}
		result = append(result, id)
	}
	*a = result
	return nil
}

// Value implements driver.Valuer for writing PostgreSQL uuid[] arrays.
func (a UUIDArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	if len(a) == 0 {
		return "{}", nil
	}

	elems := make([]string, len(a))
	for i, id := range a {
		elems[i] = id.String()
	}
	return "{" + strings.Join(elems, ",") + "}", nil
}

func parseArrayElements(s string) []string {
	var result []string
	var current strings.Builder
	inQuote := false
	escaped := false

	for _, r := range s {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		switch {
		case r == '\\':
			escaped = true
		case r == '"':
			inQuote = !inQuote
		case r == ',' && !inQuote:
			result = append(result, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	result = append(result, current.String())
	return result
}

func quoteArrayElement(s string) string {
	if s == "" || strings.ContainsAny(s, `{},"\`) || strings.ContainsAny(s, " \t\n") {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}
