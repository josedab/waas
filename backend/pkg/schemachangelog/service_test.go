package schemachangelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterSchemaAndChangelog(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	// Register v1
	v1, _, err := svc.RegisterSchema("t1", &RegisterSchemaRequest{
		EventType: "order.created",
		Version:   "1.0.0",
		Schema: map[string]interface{}{
			"id":     "string",
			"amount": "number",
			"status": "string",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", v1.Version)

	// Register v2 with breaking change (removed field)
	v2, changelog, err := svc.RegisterSchema("t1", &RegisterSchemaRequest{
		EventType: "order.created",
		Version:   "2.0.0",
		Schema: map[string]interface{}{
			"id":       "string",
			"amount":   "number",
			"currency": "string", // new field
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", v2.Version)
	require.NotNil(t, changelog)
	assert.True(t, changelog.HasBreaking) // status removed
	assert.NotEmpty(t, changelog.MigrationGuide)
}

func TestDiffSchemas(t *testing.T) {
	old := map[string]interface{}{
		"id":     "string",
		"name":   "string",
		"amount": "number",
	}
	new := map[string]interface{}{
		"id":     "string",
		"amount": "integer", // type change
		"email":  "string",  // addition
	}

	changes := diffSchemas("", old, new)

	var breaking, additions int
	for _, c := range changes {
		switch c.ChangeType {
		case ChangeBreaking:
			breaking++
		case ChangeAddition:
			additions++
		}
	}

	assert.Equal(t, 1, breaking)  // name removed
	assert.Equal(t, 1, additions) // email added
}

func TestMigrationTracking(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	// Create schema versions
	svc.RegisterSchema("t1", &RegisterSchemaRequest{
		EventType: "order.created", Version: "1.0.0",
		Schema: map[string]interface{}{"id": "string"},
	})
	_, changelog, _ := svc.RegisterSchema("t1", &RegisterSchemaRequest{
		EventType: "order.created", Version: "2.0.0",
		Schema: map[string]interface{}{"id": "string", "new_field": "string"},
	})

	require.NotNil(t, changelog)

	// Create migration tracking
	migrations, err := svc.CreateMigrationTracking("t1", changelog.ID, []string{"ep-1", "ep-2"})
	require.NoError(t, err)
	assert.Len(t, migrations, 2)
	assert.Equal(t, "pending", migrations[0].Status)

	// Acknowledge
	m, err := svc.AcknowledgeMigration(migrations[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "acknowledged", m.Status)
	assert.NotNil(t, m.AcknowledgedAt)

	// Complete
	m, err = svc.CompleteMigration(migrations[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", m.Status)
	assert.NotNil(t, m.CompletedAt)
}

func TestGenerateSummary(t *testing.T) {
	changes := []SchemaChange{
		{ChangeType: ChangeBreaking},
		{ChangeType: ChangeBreaking},
		{ChangeType: ChangeAddition},
	}
	summary := generateSummary(changes)
	assert.Contains(t, summary, "2 breaking change(s)")
	assert.Contains(t, summary, "1 addition(s)")
}
