package pluginmarket

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Marketplace tests ---

func TestMarketplaceService_CreateAndBrowse(t *testing.T) {
	svc := NewMarketplaceService()
	ctx := context.Background()

	listing, err := svc.CreateListing(ctx, &MarketplaceListing{
		DeveloperID: "dev-1",
		Name:        "Test Plugin",
		Description: "A test plugin",
		Category:    "transforms",
		Tags:        []string{"json", "transform"},
		Price:       9.99,
		Versions:    []PluginVersionV2{{Version: "1.0.0", Changelog: "Initial"}},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, listing.ID)
	assert.Equal(t, ListingStatusDraft, listing.Status)
	assert.Equal(t, "USD", listing.Currency)

	// Draft listings should not appear in browse
	results, total := svc.Browse(ctx, &MarketplaceSearchParams{})
	assert.Equal(t, 0, total)
	assert.Empty(t, results)

	// Publish and browse again
	err = svc.PublishListing(ctx, listing.ID, "dev-1")
	require.NoError(t, err)

	results, total = svc.Browse(ctx, &MarketplaceSearchParams{})
	assert.Equal(t, 1, total)
	assert.Equal(t, "Test Plugin", results[0].Name)
}

func TestMarketplaceService_SearchFilters(t *testing.T) {
	svc := NewMarketplaceService()
	ctx := context.Background()

	for _, p := range []struct {
		name, cat string
		tags      []string
	}{
		{"Slack Notifier", "notifications", []string{"slack"}},
		{"JSON Transform", "transforms", []string{"json"}},
		{"PII Redactor", "security", []string{"pii", "security"}},
	} {
		l, err := svc.CreateListing(ctx, &MarketplaceListing{
			DeveloperID: "dev-1", Name: p.name, Description: p.name + " desc",
			Category: p.cat, Tags: p.tags,
		})
		require.NoError(t, err)
		require.NoError(t, svc.PublishListing(ctx, l.ID, "dev-1"))
	}

	// Filter by category
	results, total := svc.Browse(ctx, &MarketplaceSearchParams{Category: "security"})
	assert.Equal(t, 1, total)
	assert.Equal(t, "PII Redactor", results[0].Name)

	// Filter by tag
	results, total = svc.Browse(ctx, &MarketplaceSearchParams{Tag: "slack"})
	assert.Equal(t, 1, total)
	assert.Equal(t, "Slack Notifier", results[0].Name)

	// Search by query
	results, total = svc.Search(ctx, "json")
	assert.Equal(t, 1, total)
	assert.Equal(t, "JSON Transform", results[0].Name)
}

func TestMarketplaceService_SortByRatingAndDownloads(t *testing.T) {
	svc := NewMarketplaceService()
	ctx := context.Background()

	l1, _ := svc.CreateListing(ctx, &MarketplaceListing{DeveloperID: "d", Name: "Low", Description: "low"})
	l2, _ := svc.CreateListing(ctx, &MarketplaceListing{DeveloperID: "d", Name: "High", Description: "high"})
	_ = svc.PublishListing(ctx, l1.ID, "d")
	_ = svc.PublishListing(ctx, l2.ID, "d")

	// Add reviews to affect rating
	_, _ = svc.SubmitReview(ctx, "t1", l1.ID, 2, "meh")
	_, _ = svc.SubmitReview(ctx, "t1", l2.ID, 5, "great")

	results, _ := svc.Browse(ctx, &MarketplaceSearchParams{SortBy: "rating"})
	require.Len(t, results, 2)
	assert.Equal(t, "High", results[0].Name)

	// Install to affect downloads
	_, _ = svc.Install(ctx, "tenant-a", l1.ID)
	_, _ = svc.Install(ctx, "tenant-b", l1.ID)
	_, _ = svc.Install(ctx, "tenant-c", l1.ID)
	_, _ = svc.Install(ctx, "tenant-a", l2.ID)

	results, _ = svc.Browse(ctx, &MarketplaceSearchParams{SortBy: "downloads"})
	require.Len(t, results, 2)
	assert.Equal(t, "Low", results[0].Name)
}

func TestMarketplaceService_InstallUninstall(t *testing.T) {
	svc := NewMarketplaceService()
	ctx := context.Background()

	listing, _ := svc.CreateListing(ctx, &MarketplaceListing{DeveloperID: "d", Name: "P", Description: "p"})
	_ = svc.PublishListing(ctx, listing.ID, "d")

	inst, err := svc.Install(ctx, "tenant-1", listing.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", inst.Status)

	installed := svc.GetInstalled(ctx, "tenant-1")
	assert.Len(t, installed, 1)

	// Duplicate install should fail
	_, err = svc.Install(ctx, "tenant-1", listing.ID)
	assert.Error(t, err)

	// Uninstall
	err = svc.Uninstall(ctx, "tenant-1", listing.ID)
	require.NoError(t, err)

	installed = svc.GetInstalled(ctx, "tenant-1")
	assert.Empty(t, installed)

	// Uninstall again should fail
	err = svc.Uninstall(ctx, "tenant-1", listing.ID)
	assert.Error(t, err)
}

func TestMarketplaceService_SubmitReview(t *testing.T) {
	svc := NewMarketplaceService()
	ctx := context.Background()

	listing, _ := svc.CreateListing(ctx, &MarketplaceListing{DeveloperID: "d", Name: "P", Description: "p"})
	_ = svc.PublishListing(ctx, listing.ID, "d")

	review, err := svc.SubmitReview(ctx, "t1", listing.ID, 4, "solid plugin")
	require.NoError(t, err)
	assert.Equal(t, 4, review.Rating)

	// Verify rating is updated on listing
	l, _ := svc.GetListing(ctx, listing.ID)
	assert.Equal(t, 4.0, l.Rating)

	// Invalid rating
	_, err = svc.SubmitReview(ctx, "t2", listing.ID, 0, "bad")
	assert.Error(t, err)
	_, err = svc.SubmitReview(ctx, "t2", listing.ID, 6, "bad")
	assert.Error(t, err)
}

func TestMarketplaceService_PublishErrors(t *testing.T) {
	svc := NewMarketplaceService()
	ctx := context.Background()

	listing, _ := svc.CreateListing(ctx, &MarketplaceListing{DeveloperID: "d", Name: "P", Description: "p"})

	// Wrong developer
	err := svc.PublishListing(ctx, listing.ID, "other")
	assert.Error(t, err)

	// Publish, then try again
	_ = svc.PublishListing(ctx, listing.ID, "d")
	err = svc.PublishListing(ctx, listing.ID, "d")
	assert.Error(t, err)
}

func TestMarketplaceService_CreateListingValidation(t *testing.T) {
	svc := NewMarketplaceService()
	ctx := context.Background()

	_, err := svc.CreateListing(ctx, &MarketplaceListing{DeveloperID: "d"})
	assert.Error(t, err, "name required")

	_, err = svc.CreateListing(ctx, &MarketplaceListing{Name: "P"})
	assert.Error(t, err, "developer ID required")
}

// --- SDK v2 / Sandbox tests ---

func TestPluginSandbox_Execute(t *testing.T) {
	sandbox := NewPluginSandbox(DefaultSandboxConfig())
	ctx := context.Background()

	result, err := sandbox.Execute(ctx, `(function() { return {sum: input.a + input.b}; })()`, map[string]interface{}{"a": 3, "b": 4})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(7), result.Output["sum"])
}

func TestPluginSandbox_Timeout(t *testing.T) {
	sandbox := NewPluginSandbox(SandboxConfig{TimeoutMs: 50, MaxMemoryKB: 1024})
	ctx := context.Background()

	result, err := sandbox.Execute(ctx, `while(true) {}`, nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "timeout")
}

func TestPluginSandbox_ErrorIsolation(t *testing.T) {
	sandbox := NewPluginSandbox(DefaultSandboxConfig())
	ctx := context.Background()

	result, err := sandbox.Execute(ctx, `throw new Error("boom")`, nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "boom")
}

func TestPluginTestRunner_RunHook(t *testing.T) {
	manifest := &PluginManifest{
		Name:    "test-plugin",
		Version: "1.0.0",
		Author:  "tester",
		Hooks:   []PluginHookTypeV2{HookTransformV2, HookValidateV2},
	}
	runner := NewPluginTestRunner(manifest, DefaultSandboxConfig())
	ctx := context.Background()

	event := NewMockWebhookEvent("order.created", map[string]interface{}{"amount": 100})
	result, err := runner.RunHook(ctx, HookTransformV2, `(function(){ return {transformed: true, amount: input.payload.amount * 2}; })()`, event)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, true, result.Output["transformed"])

	// Unsupported hook
	_, err = runner.RunHook(ctx, HookBeforeDelivery, `true`, event)
	assert.Error(t, err)
}

func TestPluginManifest(t *testing.T) {
	manifest := PluginManifest{
		Name:        "my-plugin",
		Version:     "2.0.0",
		Description: "A great plugin",
		Author:      "dev",
		Permissions: []string{"read:webhooks"},
		Hooks:       []PluginHookTypeV2{HookBeforeDelivery, HookAfterDelivery},
	}
	assert.Equal(t, "my-plugin", manifest.Name)
	assert.Len(t, manifest.Hooks, 2)
	assert.Contains(t, manifest.Permissions, "read:webhooks")
}

// --- Monetization tests ---

func TestMonetizationService_RegisterDeveloper(t *testing.T) {
	svc := NewMonetizationService()
	ctx := context.Background()

	acc, err := svc.RegisterDeveloper(ctx, "dev-1", "dev@example.com", "Dev One", "acct_stripe_123")
	require.NoError(t, err)
	assert.NotEmpty(t, acc.ID)
	assert.Equal(t, "acct_stripe_123", acc.StripeConnectID)
	assert.Equal(t, DefaultDeveloperSharePct, acc.RevenueShare.DeveloperPct)

	// Duplicate registration
	_, err = svc.RegisterDeveloper(ctx, "dev-1", "dev@example.com", "Dev One", "acct_stripe_456")
	assert.Error(t, err)

	// Missing fields
	_, err = svc.RegisterDeveloper(ctx, "", "a@b.com", "X", "acct_1")
	assert.Error(t, err)
	_, err = svc.RegisterDeveloper(ctx, "dev-2", "a@b.com", "X", "")
	assert.Error(t, err)
}

func TestMonetizationService_RecordPurchaseAndEarnings(t *testing.T) {
	svc := NewMonetizationService()
	ctx := context.Background()

	_, _ = svc.RegisterDeveloper(ctx, "dev-1", "dev@example.com", "Dev", "acct_1")

	tx, err := svc.RecordPurchase(ctx, "listing-1", "tenant-1", "dev-1", 100.0, "USD")
	require.NoError(t, err)
	assert.Equal(t, TransactionCompleted, tx.Status)
	assert.Equal(t, 70.0, tx.DeveloperAmount)
	assert.Equal(t, 30.0, tx.PlatformAmount)

	dashboard, err := svc.GetEarnings(ctx, "dev-1")
	require.NoError(t, err)
	assert.Equal(t, 70.0, dashboard.TotalEarnings)
	assert.Equal(t, 70.0, dashboard.PendingEarnings)
	assert.Equal(t, 0.0, dashboard.PaidEarnings)
	assert.Len(t, dashboard.Transactions, 1)
}

func TestMonetizationService_ProcessPayout(t *testing.T) {
	svc := NewMonetizationService()
	ctx := context.Background()

	_, _ = svc.RegisterDeveloper(ctx, "dev-1", "dev@example.com", "Dev", "acct_1")
	_, _ = svc.RecordPurchase(ctx, "listing-1", "tenant-1", "dev-1", 100.0, "USD")

	payout, err := svc.ProcessPayout(ctx, "dev-1")
	require.NoError(t, err)
	assert.Equal(t, PayoutPaid, payout.Status)
	assert.Equal(t, 70.0, payout.Amount)
	assert.NotNil(t, payout.PaidAt)

	// After payout, pending should be 0
	dashboard, _ := svc.GetEarnings(ctx, "dev-1")
	assert.Equal(t, 0.0, dashboard.PendingEarnings)
	assert.Equal(t, 70.0, dashboard.PaidEarnings)
	assert.Len(t, dashboard.Payouts, 1)

	// No pending earnings to pay out
	_, err = svc.ProcessPayout(ctx, "dev-1")
	assert.Error(t, err)
}

func TestMonetizationService_PurchaseErrors(t *testing.T) {
	svc := NewMonetizationService()
	ctx := context.Background()

	// Unknown developer
	_, err := svc.RecordPurchase(ctx, "l1", "t1", "unknown", 10.0, "USD")
	assert.Error(t, err)

	// Non-positive amount
	_, _ = svc.RegisterDeveloper(ctx, "dev-1", "a@b.com", "D", "acct_1")
	_, err = svc.RecordPurchase(ctx, "l1", "t1", "dev-1", 0, "USD")
	assert.Error(t, err)
}

func TestMonetizationService_GetEarningsUnknown(t *testing.T) {
	svc := NewMonetizationService()
	_, err := svc.GetEarnings(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestDefaultRevenueShare(t *testing.T) {
	rs := DefaultRevenueShare()
	assert.Equal(t, 70, rs.DeveloperPct)
	assert.Equal(t, 30, rs.PlatformPct)
}
