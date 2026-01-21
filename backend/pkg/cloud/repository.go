package cloud

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Repository defines the cloud billing data access interface
type Repository interface {
	// Subscriptions
	CreateSubscription(ctx context.Context, sub *Subscription) error
	GetSubscription(ctx context.Context, id string) (*Subscription, error)
	GetSubscriptionByTenant(ctx context.Context, tenantID string) (*Subscription, error)
	UpdateSubscription(ctx context.Context, sub *Subscription) error
	ListSubscriptions(ctx context.Context, status SubscriptionStatus, limit int) ([]*Subscription, error)

	// Usage
	GetUsage(ctx context.Context, tenantID, period string) (*UsageRecord, error)
	IncrementUsage(ctx context.Context, tenantID string, field string, delta int64) error
	ListUsage(ctx context.Context, tenantID string, limit int) ([]*UsageRecord, error)

	// Invoices
	CreateInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoice(ctx context.Context, id string) (*Invoice, error)
	ListInvoices(ctx context.Context, tenantID string, limit int) ([]*Invoice, error)
	UpdateInvoice(ctx context.Context, invoice *Invoice) error

	// Customers
	CreateCustomer(ctx context.Context, customer *Customer) error
	GetCustomer(ctx context.Context, tenantID string) (*Customer, error)
	UpdateCustomer(ctx context.Context, customer *Customer) error

	// Payment Methods
	CreatePaymentMethod(ctx context.Context, pm *PaymentMethod) error
	ListPaymentMethods(ctx context.Context, tenantID string) ([]*PaymentMethod, error)
	DeletePaymentMethod(ctx context.Context, id string) error
	SetDefaultPaymentMethod(ctx context.Context, tenantID, paymentMethodID string) error

	// Team Members
	CreateTeamMember(ctx context.Context, member *TeamMember) error
	GetTeamMember(ctx context.Context, id string) (*TeamMember, error)
	ListTeamMembers(ctx context.Context, tenantID string) ([]*TeamMember, error)
	UpdateTeamMember(ctx context.Context, member *TeamMember) error
	DeleteTeamMember(ctx context.Context, id string) error

	// Audit Logs
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	ListAuditLogs(ctx context.Context, tenantID string, limit, offset int) ([]*AuditLog, error)
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateSubscription creates a subscription
func (r *PostgresRepository) CreateSubscription(ctx context.Context, sub *Subscription) error {
	query := `
		INSERT INTO subscriptions (id, tenant_id, plan_id, status, billing_cycle, 
		current_period_start, current_period_end, trial_end, cancel_at_period_end,
		stripe_subscription_id, stripe_customer_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.TenantID, sub.PlanID, sub.Status, sub.BillingCycle,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.TrialEnd, sub.CancelAtPeriodEnd,
		sub.StripeSubscriptionID, sub.StripeCustomerID, sub.CreatedAt, sub.UpdatedAt)

	return err
}

// GetSubscription retrieves a subscription by ID
func (r *PostgresRepository) GetSubscription(ctx context.Context, id string) (*Subscription, error) {
	query := `
		SELECT id, tenant_id, plan_id, status, billing_cycle, current_period_start,
		current_period_end, trial_end, cancel_at_period_end, canceled_at,
		stripe_subscription_id, stripe_customer_id, created_at, updated_at
		FROM subscriptions WHERE id = $1`

	var sub Subscription
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Status, &sub.BillingCycle,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEnd, &sub.CancelAtPeriodEnd,
		&sub.CanceledAt, &sub.StripeSubscriptionID, &sub.StripeCustomerID,
		&sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// GetSubscriptionByTenant retrieves a subscription by tenant ID
func (r *PostgresRepository) GetSubscriptionByTenant(ctx context.Context, tenantID string) (*Subscription, error) {
	query := `
		SELECT id, tenant_id, plan_id, status, billing_cycle, current_period_start,
		current_period_end, trial_end, cancel_at_period_end, canceled_at,
		stripe_subscription_id, stripe_customer_id, created_at, updated_at
		FROM subscriptions WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1`

	var sub Subscription
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Status, &sub.BillingCycle,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEnd, &sub.CancelAtPeriodEnd,
		&sub.CanceledAt, &sub.StripeSubscriptionID, &sub.StripeCustomerID,
		&sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// UpdateSubscription updates a subscription
func (r *PostgresRepository) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	query := `
		UPDATE subscriptions SET plan_id = $2, status = $3, billing_cycle = $4,
		current_period_start = $5, current_period_end = $6, trial_end = $7,
		cancel_at_period_end = $8, canceled_at = $9, stripe_subscription_id = $10,
		stripe_customer_id = $11, updated_at = $12
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.PlanID, sub.Status, sub.BillingCycle,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.TrialEnd,
		sub.CancelAtPeriodEnd, sub.CanceledAt, sub.StripeSubscriptionID,
		sub.StripeCustomerID, time.Now())

	return err
}

// ListSubscriptions lists subscriptions by status
func (r *PostgresRepository) ListSubscriptions(ctx context.Context, status SubscriptionStatus, limit int) ([]*Subscription, error) {
	query := `
		SELECT id, tenant_id, plan_id, status, billing_cycle, current_period_start,
		current_period_end, trial_end, cancel_at_period_end, canceled_at,
		stripe_subscription_id, stripe_customer_id, created_at, updated_at
		FROM subscriptions WHERE status = $1 ORDER BY created_at DESC LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Status, &sub.BillingCycle,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEnd, &sub.CancelAtPeriodEnd,
			&sub.CanceledAt, &sub.StripeSubscriptionID, &sub.StripeCustomerID,
			&sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, &sub)
	}

	return subs, rows.Err()
}

// GetUsage retrieves usage for a period
func (r *PostgresRepository) GetUsage(ctx context.Context, tenantID, period string) (*UsageRecord, error) {
	query := `
		SELECT id, tenant_id, period, webhooks_sent, webhooks_received, successful_deliveries,
		failed_deliveries, total_bytes, api_requests, transform_executions, storage_bytes, updated_at
		FROM usage_records WHERE tenant_id = $1 AND period = $2`

	var usage UsageRecord
	err := r.db.QueryRowContext(ctx, query, tenantID, period).Scan(
		&usage.ID, &usage.TenantID, &usage.Period, &usage.WebhooksSent, &usage.WebhooksReceived,
		&usage.SuccessfulDeliveries, &usage.FailedDeliveries, &usage.TotalBytes,
		&usage.APIRequests, &usage.TransformExecutions, &usage.StorageBytes, &usage.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &usage, nil
}

// IncrementUsage increments a usage field
func (r *PostgresRepository) IncrementUsage(ctx context.Context, tenantID string, field string, delta int64) error {
	period := time.Now().Format("2006-01")

	query := `
		INSERT INTO usage_records (id, tenant_id, period, ` + field + `, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		ON CONFLICT (tenant_id, period) DO UPDATE
		SET ` + field + ` = usage_records.` + field + ` + $3, updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query, tenantID, period, delta)
	return err
}

// ListUsage lists usage records for a tenant
func (r *PostgresRepository) ListUsage(ctx context.Context, tenantID string, limit int) ([]*UsageRecord, error) {
	query := `
		SELECT id, tenant_id, period, webhooks_sent, webhooks_received, successful_deliveries,
		failed_deliveries, total_bytes, api_requests, transform_executions, storage_bytes, updated_at
		FROM usage_records WHERE tenant_id = $1 ORDER BY period DESC LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*UsageRecord
	for rows.Next() {
		var usage UsageRecord
		if err := rows.Scan(
			&usage.ID, &usage.TenantID, &usage.Period, &usage.WebhooksSent, &usage.WebhooksReceived,
			&usage.SuccessfulDeliveries, &usage.FailedDeliveries, &usage.TotalBytes,
			&usage.APIRequests, &usage.TransformExecutions, &usage.StorageBytes, &usage.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, &usage)
	}

	return records, rows.Err()
}

// CreateInvoice creates an invoice
func (r *PostgresRepository) CreateInvoice(ctx context.Context, invoice *Invoice) error {
	lineItems, _ := json.Marshal(invoice.LineItems)

	query := `
		INSERT INTO invoices (id, tenant_id, subscription_id, number, status, currency,
		subtotal, tax, total, amount_paid, amount_due, line_items, period_start, period_end,
		due_date, stripe_invoice_id, pdf_url, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`

	_, err := r.db.ExecContext(ctx, query,
		invoice.ID, invoice.TenantID, invoice.SubscriptionID, invoice.Number, invoice.Status,
		invoice.Currency, invoice.Subtotal, invoice.Tax, invoice.Total, invoice.AmountPaid,
		invoice.AmountDue, lineItems, invoice.PeriodStart, invoice.PeriodEnd, invoice.DueDate,
		invoice.StripeInvoiceID, invoice.PDFUrl, invoice.CreatedAt)

	return err
}

// GetInvoice retrieves an invoice by ID
func (r *PostgresRepository) GetInvoice(ctx context.Context, id string) (*Invoice, error) {
	query := `
		SELECT id, tenant_id, subscription_id, number, status, currency, subtotal, tax,
		total, amount_paid, amount_due, line_items, period_start, period_end, due_date,
		paid_at, stripe_invoice_id, pdf_url, created_at
		FROM invoices WHERE id = $1`

	var invoice Invoice
	var lineItems []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&invoice.ID, &invoice.TenantID, &invoice.SubscriptionID, &invoice.Number,
		&invoice.Status, &invoice.Currency, &invoice.Subtotal, &invoice.Tax, &invoice.Total,
		&invoice.AmountPaid, &invoice.AmountDue, &lineItems, &invoice.PeriodStart,
		&invoice.PeriodEnd, &invoice.DueDate, &invoice.PaidAt, &invoice.StripeInvoiceID,
		&invoice.PDFUrl, &invoice.CreatedAt)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(lineItems, &invoice.LineItems)
	return &invoice, nil
}

// ListInvoices lists invoices for a tenant
func (r *PostgresRepository) ListInvoices(ctx context.Context, tenantID string, limit int) ([]*Invoice, error) {
	query := `
		SELECT id, tenant_id, subscription_id, number, status, currency, subtotal, tax,
		total, amount_paid, amount_due, line_items, period_start, period_end, due_date,
		paid_at, stripe_invoice_id, pdf_url, created_at
		FROM invoices WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*Invoice
	for rows.Next() {
		var invoice Invoice
		var lineItems []byte

		if err := rows.Scan(
			&invoice.ID, &invoice.TenantID, &invoice.SubscriptionID, &invoice.Number,
			&invoice.Status, &invoice.Currency, &invoice.Subtotal, &invoice.Tax, &invoice.Total,
			&invoice.AmountPaid, &invoice.AmountDue, &lineItems, &invoice.PeriodStart,
			&invoice.PeriodEnd, &invoice.DueDate, &invoice.PaidAt, &invoice.StripeInvoiceID,
			&invoice.PDFUrl, &invoice.CreatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal(lineItems, &invoice.LineItems)
		invoices = append(invoices, &invoice)
	}

	return invoices, rows.Err()
}

// UpdateInvoice updates an invoice
func (r *PostgresRepository) UpdateInvoice(ctx context.Context, invoice *Invoice) error {
	query := `
		UPDATE invoices SET status = $2, amount_paid = $3, amount_due = $4, paid_at = $5
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		invoice.ID, invoice.Status, invoice.AmountPaid, invoice.AmountDue, invoice.PaidAt)

	return err
}

// CreateCustomer creates a customer
func (r *PostgresRepository) CreateCustomer(ctx context.Context, customer *Customer) error {
	query := `
		INSERT INTO customers (id, tenant_id, email, name, company, address_line1, address_line2,
		city, state, postal_code, country, tax_id, stripe_customer_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := r.db.ExecContext(ctx, query,
		customer.ID, customer.TenantID, customer.Email, customer.Name, customer.Company,
		customer.AddressLine1, customer.AddressLine2, customer.City, customer.State,
		customer.PostalCode, customer.Country, customer.TaxID, customer.StripeCustomerID,
		customer.CreatedAt, customer.UpdatedAt)

	return err
}

// GetCustomer retrieves a customer by tenant ID
func (r *PostgresRepository) GetCustomer(ctx context.Context, tenantID string) (*Customer, error) {
	query := `
		SELECT id, tenant_id, email, name, company, address_line1, address_line2,
		city, state, postal_code, country, tax_id, stripe_customer_id, created_at, updated_at
		FROM customers WHERE tenant_id = $1`

	var customer Customer
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&customer.ID, &customer.TenantID, &customer.Email, &customer.Name, &customer.Company,
		&customer.AddressLine1, &customer.AddressLine2, &customer.City, &customer.State,
		&customer.PostalCode, &customer.Country, &customer.TaxID, &customer.StripeCustomerID,
		&customer.CreatedAt, &customer.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// UpdateCustomer updates a customer
func (r *PostgresRepository) UpdateCustomer(ctx context.Context, customer *Customer) error {
	query := `
		UPDATE customers SET email = $2, name = $3, company = $4, address_line1 = $5,
		address_line2 = $6, city = $7, state = $8, postal_code = $9, country = $10,
		tax_id = $11, stripe_customer_id = $12, updated_at = $13
		WHERE tenant_id = $1`

	_, err := r.db.ExecContext(ctx, query,
		customer.TenantID, customer.Email, customer.Name, customer.Company,
		customer.AddressLine1, customer.AddressLine2, customer.City, customer.State,
		customer.PostalCode, customer.Country, customer.TaxID, customer.StripeCustomerID,
		time.Now())

	return err
}

// CreatePaymentMethod creates a payment method
func (r *PostgresRepository) CreatePaymentMethod(ctx context.Context, pm *PaymentMethod) error {
	query := `
		INSERT INTO payment_methods (id, tenant_id, type, is_default, card_brand, card_last4,
		card_exp_month, card_exp_year, stripe_payment_method_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		pm.ID, pm.TenantID, pm.Type, pm.IsDefault, pm.CardBrand, pm.CardLast4,
		pm.CardExpMonth, pm.CardExpYear, pm.StripePaymentMethodID, pm.CreatedAt)

	return err
}

// ListPaymentMethods lists payment methods for a tenant
func (r *PostgresRepository) ListPaymentMethods(ctx context.Context, tenantID string) ([]*PaymentMethod, error) {
	query := `
		SELECT id, tenant_id, type, is_default, card_brand, card_last4, card_exp_month,
		card_exp_year, stripe_payment_method_id, created_at
		FROM payment_methods WHERE tenant_id = $1 ORDER BY is_default DESC, created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var methods []*PaymentMethod
	for rows.Next() {
		var pm PaymentMethod
		if err := rows.Scan(
			&pm.ID, &pm.TenantID, &pm.Type, &pm.IsDefault, &pm.CardBrand, &pm.CardLast4,
			&pm.CardExpMonth, &pm.CardExpYear, &pm.StripePaymentMethodID, &pm.CreatedAt); err != nil {
			return nil, err
		}
		methods = append(methods, &pm)
	}

	return methods, rows.Err()
}

// DeletePaymentMethod deletes a payment method
func (r *PostgresRepository) DeletePaymentMethod(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM payment_methods WHERE id = $1", id)
	return err
}

// SetDefaultPaymentMethod sets the default payment method
func (r *PostgresRepository) SetDefaultPaymentMethod(ctx context.Context, tenantID, paymentMethodID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing default
	_, err = tx.ExecContext(ctx, "UPDATE payment_methods SET is_default = FALSE WHERE tenant_id = $1", tenantID)
	if err != nil {
		return err
	}

	// Set new default
	_, err = tx.ExecContext(ctx, "UPDATE payment_methods SET is_default = TRUE WHERE id = $1 AND tenant_id = $2", paymentMethodID, tenantID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// CreateTeamMember creates a team member
func (r *PostgresRepository) CreateTeamMember(ctx context.Context, member *TeamMember) error {
	query := `
		INSERT INTO team_members (id, tenant_id, user_id, email, name, role, status, invited_by, invited_at, joined_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		member.ID, member.TenantID, member.UserID, member.Email, member.Name,
		member.Role, member.Status, member.InvitedBy, member.InvitedAt, member.JoinedAt)

	return err
}

// GetTeamMember retrieves a team member
func (r *PostgresRepository) GetTeamMember(ctx context.Context, id string) (*TeamMember, error) {
	query := `
		SELECT id, tenant_id, user_id, email, name, role, status, invited_by, invited_at, joined_at
		FROM team_members WHERE id = $1`

	var member TeamMember
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&member.ID, &member.TenantID, &member.UserID, &member.Email, &member.Name,
		&member.Role, &member.Status, &member.InvitedBy, &member.InvitedAt, &member.JoinedAt)

	if err != nil {
		return nil, err
	}
	return &member, nil
}

// ListTeamMembers lists team members for a tenant
func (r *PostgresRepository) ListTeamMembers(ctx context.Context, tenantID string) ([]*TeamMember, error) {
	query := `
		SELECT id, tenant_id, user_id, email, name, role, status, invited_by, invited_at, joined_at
		FROM team_members WHERE tenant_id = $1 ORDER BY role, name`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*TeamMember
	for rows.Next() {
		var member TeamMember
		if err := rows.Scan(
			&member.ID, &member.TenantID, &member.UserID, &member.Email, &member.Name,
			&member.Role, &member.Status, &member.InvitedBy, &member.InvitedAt, &member.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, &member)
	}

	return members, rows.Err()
}

// UpdateTeamMember updates a team member
func (r *PostgresRepository) UpdateTeamMember(ctx context.Context, member *TeamMember) error {
	query := `
		UPDATE team_members SET email = $2, name = $3, role = $4, status = $5, joined_at = $6
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		member.ID, member.Email, member.Name, member.Role, member.Status, member.JoinedAt)

	return err
}

// DeleteTeamMember deletes a team member
func (r *PostgresRepository) DeleteTeamMember(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM team_members WHERE id = $1", id)
	return err
}

// CreateAuditLog creates an audit log entry
func (r *PostgresRepository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	details, _ := json.Marshal(log.Details)

	query := `
		INSERT INTO audit_logs (id, tenant_id, user_id, action, resource, resource_id,
		ip_address, user_agent, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.TenantID, log.UserID, log.Action, log.Resource, log.ResourceID,
		log.IPAddress, log.UserAgent, details, log.CreatedAt)

	return err
}

// ListAuditLogs lists audit logs for a tenant
func (r *PostgresRepository) ListAuditLogs(ctx context.Context, tenantID string, limit, offset int) ([]*AuditLog, error) {
	query := `
		SELECT id, tenant_id, user_id, action, resource, resource_id, ip_address,
		user_agent, details, created_at
		FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		var log AuditLog
		var details []byte

		if err := rows.Scan(
			&log.ID, &log.TenantID, &log.UserID, &log.Action, &log.Resource, &log.ResourceID,
			&log.IPAddress, &log.UserAgent, &details, &log.CreatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal(details, &log.Details)
		logs = append(logs, &log)
	}

	return logs, rows.Err()
}
