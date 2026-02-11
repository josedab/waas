package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPAllowlistService_AddEntry_ValidIPv4(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	entry, err := svc.AddEntry(ctx, "tenant1", "ep1", "203.0.113.1", "test server")
	require.NoError(t, err)
	assert.NotEmpty(t, entry.ID)
	assert.Equal(t, "203.0.113.1/32", entry.CIDR) // single IP normalized to /32
	assert.Equal(t, "test server", entry.Label)
	assert.Equal(t, "tenant1", entry.TenantID)
	assert.Equal(t, "ep1", entry.EndpointID)
}

func TestIPAllowlistService_AddEntry_ValidIPv6(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	entry, err := svc.AddEntry(ctx, "tenant1", "ep1", "2001:db8::1", "ipv6 server")
	require.NoError(t, err)
	assert.Equal(t, "2001:db8::1/128", entry.CIDR) // single IPv6 normalized to /128
}

func TestIPAllowlistService_AddEntry_ValidCIDR(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	entry, err := svc.AddEntry(ctx, "tenant1", "ep1", "10.0.0.0/8", "private range")
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.0/8", entry.CIDR)
}

func TestIPAllowlistService_AddEntry_InvalidCIDR(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	_, err := svc.AddEntry(ctx, "tenant1", "ep1", "invalid-ip", "bad entry")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CIDR or IP")
}

func TestIPAllowlistService_AddEntry_InvalidCIDRRange(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	_, err := svc.AddEntry(ctx, "tenant1", "ep1", "10.0.0.0/999", "bad cidr")
	assert.Error(t, err)
}

func TestIPAllowlistService_CheckIP_AllowlistMode(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	// Add allowed IP
	_, err := svc.AddEntry(ctx, "tenant1", "ep1", "203.0.113.0/24", "allowed range")
	require.NoError(t, err)

	// IP in range should be allowed
	allowed, err := svc.CheckIP(ctx, "ep1", "203.0.113.50")
	require.NoError(t, err)
	assert.True(t, allowed)

	// IP outside range should be denied
	allowed, err = svc.CheckIP(ctx, "ep1", "198.51.100.1")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestIPAllowlistService_CheckIP_DenylistMode(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	// Set up denylist mode
	config := &IPAllowlistConfig{
		EndpointID: "ep1",
		Enabled:    true,
		Mode:       "denylist",
		Entries: []IPAllowlistEntry{
			{ID: "e1", CIDR: "192.168.0.0/16"},
		},
	}
	require.NoError(t, svc.SetConfig(ctx, config))

	// IP in denylist should be denied
	allowed, err := svc.CheckIP(ctx, "ep1", "192.168.1.1")
	require.NoError(t, err)
	assert.False(t, allowed)

	// IP not in denylist should be allowed
	allowed, err = svc.CheckIP(ctx, "ep1", "8.8.8.8")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIPAllowlistService_CheckIP_NoConfig(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	// No config = allow all
	allowed, err := svc.CheckIP(ctx, "unknown-endpoint", "1.2.3.4")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIPAllowlistService_CheckIP_DisabledConfig(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	config := &IPAllowlistConfig{
		EndpointID: "ep1",
		Enabled:    false,
		Mode:       "allowlist",
		Entries:    []IPAllowlistEntry{{ID: "e1", CIDR: "10.0.0.0/8"}},
	}
	require.NoError(t, svc.SetConfig(ctx, config))

	// Disabled = allow all
	allowed, err := svc.CheckIP(ctx, "ep1", "8.8.8.8")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIPAllowlistService_CheckIP_InvalidIP(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	_, err := svc.AddEntry(ctx, "t1", "ep1", "10.0.0.0/8", "test")
	require.NoError(t, err)

	_, err = svc.CheckIP(ctx, "ep1", "not-an-ip")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid IP address")
}

func TestIPAllowlistService_CheckIP_ExactIPMatch(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	_, err := svc.AddEntry(ctx, "t1", "ep1", "1.2.3.4", "exact ip")
	require.NoError(t, err)

	allowed, err := svc.CheckIP(ctx, "ep1", "1.2.3.4")
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = svc.CheckIP(ctx, "ep1", "1.2.3.5")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestIPAllowlistService_RemoveEntry(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	entry, err := svc.AddEntry(ctx, "t1", "ep1", "10.0.0.1", "to remove")
	require.NoError(t, err)

	err = svc.RemoveEntry(ctx, "ep1", entry.ID)
	require.NoError(t, err)

	// After removal, should deny in allowlist mode (empty allowlist)
	allowed, err := svc.CheckIP(ctx, "ep1", "10.0.0.1")
	require.NoError(t, err)
	assert.False(t, allowed) // empty allowlist denies all
}

func TestIPAllowlistService_RemoveEntry_NotFound(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	_, err := svc.AddEntry(ctx, "t1", "ep1", "10.0.0.1", "test")
	require.NoError(t, err)

	err = svc.RemoveEntry(ctx, "ep1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "entry not found")
}

func TestIPAllowlistService_RemoveEntry_EndpointNotFound(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	err := svc.RemoveEntry(ctx, "nonexistent-ep", "some-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint not found")
}

func TestIPAllowlistService_SetConfig_EmptyEndpointID(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	err := svc.SetConfig(ctx, &IPAllowlistConfig{EndpointID: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint_id is required")
}

func TestIPAllowlistService_SetConfig_InvalidEntry(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	config := &IPAllowlistConfig{
		EndpointID: "ep1",
		Enabled:    true,
		Mode:       "allowlist",
		Entries: []IPAllowlistEntry{
			{ID: "e1", CIDR: "invalid"},
		},
	}
	err := svc.SetConfig(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CIDR or IP")
}

func TestIPAllowlistService_GetConfig_ExistingEndpoint(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	_, err := svc.AddEntry(ctx, "t1", "ep1", "10.0.0.0/8", "test")
	require.NoError(t, err)

	config, err := svc.GetConfig(ctx, "ep1")
	require.NoError(t, err)
	assert.Equal(t, "ep1", config.EndpointID)
	assert.True(t, config.Enabled)
	assert.Len(t, config.Entries, 1)
}

func TestIPAllowlistService_GetConfig_NonExistentEndpoint(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	config, err := svc.GetConfig(ctx, "unknown")
	require.NoError(t, err)
	assert.False(t, config.Enabled)
	assert.Empty(t, config.Entries)
}

func TestIPAllowlistService_CheckIP_OverlappingCIDRs(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	// Add overlapping ranges
	_, err := svc.AddEntry(ctx, "t1", "ep1", "10.0.0.0/8", "broad")
	require.NoError(t, err)
	_, err = svc.AddEntry(ctx, "t1", "ep1", "10.1.0.0/16", "narrow")
	require.NoError(t, err)

	// IP matching both should be allowed
	allowed, err := svc.CheckIP(ctx, "ep1", "10.1.2.3")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIPAllowlistService_CheckIP_IPv4MappedIPv6(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	_, err := svc.AddEntry(ctx, "t1", "ep1", "10.0.0.0/8", "ipv4 range")
	require.NoError(t, err)

	// IPv4-mapped IPv6 address ::ffff:10.0.0.1 should match the IPv4 CIDR
	allowed, err := svc.CheckIP(ctx, "ep1", "::ffff:10.0.0.1")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIPAllowlistService_EmptyAllowlist_DeniesAll(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	// Set enabled allowlist with no entries
	config := &IPAllowlistConfig{
		EndpointID: "ep1",
		Enabled:    true,
		Mode:       "allowlist",
		Entries:    []IPAllowlistEntry{},
	}
	require.NoError(t, svc.SetConfig(ctx, config))

	// Empty allowlist should deny all
	allowed, err := svc.CheckIP(ctx, "ep1", "8.8.8.8")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestIPAllowlistService_EmptyDenylist_AllowsAll(t *testing.T) {
	t.Parallel()
	svc := NewIPAllowlistService()
	ctx := context.Background()

	// Set enabled denylist with no entries
	config := &IPAllowlistConfig{
		EndpointID: "ep1",
		Enabled:    true,
		Mode:       "denylist",
		Entries:    []IPAllowlistEntry{},
	}
	require.NoError(t, svc.SetConfig(ctx, config))

	// Empty denylist should allow all
	allowed, err := svc.CheckIP(ctx, "ep1", "8.8.8.8")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIsValidCIDROrIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		valid bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.0/8", true},
		{"2001:db8::1", true},
		{"2001:db8::/32", true},
		{"invalid", false},
		{"10.0.0.0/999", false},
		{"", false},
		{"256.256.256.256", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, isValidCIDROrIP(tt.input))
		})
	}
}
