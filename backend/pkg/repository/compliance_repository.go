package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"webhook-platform/pkg/models"
)

// ComplianceRepository handles compliance data persistence
type ComplianceRepository interface {
	// Profile operations
	CreateProfile(ctx context.Context, profile *models.ComplianceProfile) error
	GetProfile(ctx context.Context, id uuid.UUID) (*models.ComplianceProfile, error)
	GetProfilesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ComplianceProfile, error)
	GetProfileByFramework(ctx context.Context, tenantID uuid.UUID, framework string) (*models.ComplianceProfile, error)
	UpdateProfile(ctx context.Context, profile *models.ComplianceProfile) error
	DeleteProfile(ctx context.Context, id uuid.UUID) error

	// Retention policy operations
	CreateRetentionPolicy(ctx context.Context, policy *models.DataRetentionPolicy) error
	GetRetentionPolicy(ctx context.Context, id uuid.UUID) (*models.DataRetentionPolicy, error)
	GetRetentionPoliciesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.DataRetentionPolicy, error)
	GetDueRetentionPolicies(ctx context.Context) ([]*models.DataRetentionPolicy, error)
	UpdateRetentionPolicyExecution(ctx context.Context, id uuid.UUID) error

	// PII pattern operations
	CreatePIIPattern(ctx context.Context, pattern *models.PIIDetectionPattern) error
	GetPIIPatterns(ctx context.Context, tenantID *uuid.UUID) ([]*models.PIIDetectionPattern, error)
	DeletePIIPattern(ctx context.Context, id uuid.UUID) error

	// PII detection operations
	CreatePIIDetection(ctx context.Context, detection *models.PIIDetection) error
	GetPIIDetectionsBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) ([]*models.PIIDetection, error)
	GetPIIDetectionsByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.PIIDetection, error)
	CountPIIDetectionsToday(ctx context.Context, tenantID uuid.UUID) (int, error)

	// Audit log operations
	CreateAuditLog(ctx context.Context, log *models.ComplianceAuditLog) error
	QueryAuditLogs(ctx context.Context, query *models.AuditLogQuery) ([]*models.ComplianceAuditLog, error)
	GetAuditLogsByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*models.ComplianceAuditLog, error)

	// Report operations
	CreateReport(ctx context.Context, report *models.ComplianceReport) error
	GetReport(ctx context.Context, id uuid.UUID) (*models.ComplianceReport, error)
	GetReportsByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.ComplianceReport, error)
	UpdateReportStatus(ctx context.Context, id uuid.UUID, status string, reportData map[string]interface{}, artifactURL string) error

	// Finding operations
	CreateFinding(ctx context.Context, finding *models.ComplianceFinding) error
	GetFindingsByReport(ctx context.Context, reportID uuid.UUID) ([]*models.ComplianceFinding, error)
	GetOpenFindingsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ComplianceFinding, error)
	UpdateFindingStatus(ctx context.Context, id uuid.UUID, status string) error
	CountFindingsBySeverity(ctx context.Context, tenantID uuid.UUID) (map[string]int, error)

	// DSR operations
	CreateDSR(ctx context.Context, dsr *models.DataSubjectRequest) error
	GetDSR(ctx context.Context, id uuid.UUID) (*models.DataSubjectRequest, error)
	GetDSRsByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]*models.DataSubjectRequest, error)
	UpdateDSRStatus(ctx context.Context, id uuid.UUID, status string, responseData map[string]interface{}) error
	CountPendingDSRs(ctx context.Context, tenantID uuid.UUID) (int, error)
}

// PostgresComplianceRepository implements ComplianceRepository
type PostgresComplianceRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresComplianceRepository creates a new compliance repository
func NewPostgresComplianceRepository(pool *pgxpool.Pool) *PostgresComplianceRepository {
	return &PostgresComplianceRepository{pool: pool}
}

// Profile operations

func (r *PostgresComplianceRepository) CreateProfile(ctx context.Context, profile *models.ComplianceProfile) error {
	query := `
		INSERT INTO compliance_profiles (
			id, tenant_id, name, framework, description, enabled, settings, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}

	settingsJSON, _ := json.Marshal(profile.Settings)

	_, err := r.pool.Exec(ctx, query,
		profile.ID, profile.TenantID, profile.Name, profile.Framework,
		profile.Description, profile.Enabled, settingsJSON,
	)

	return err
}

func (r *PostgresComplianceRepository) GetProfile(ctx context.Context, id uuid.UUID) (*models.ComplianceProfile, error) {
	query := `
		SELECT id, tenant_id, name, framework, description, enabled, settings, created_at, updated_at
		FROM compliance_profiles WHERE id = $1
	`

	profile := &models.ComplianceProfile{}
	var settingsJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&profile.ID, &profile.TenantID, &profile.Name, &profile.Framework,
		&profile.Description, &profile.Enabled, &settingsJSON,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("profile not found: %w", err)
	}

	json.Unmarshal(settingsJSON, &profile.Settings)
	return profile, nil
}

func (r *PostgresComplianceRepository) GetProfilesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ComplianceProfile, error) {
	query := `
		SELECT id, tenant_id, name, framework, description, enabled, settings, created_at, updated_at
		FROM compliance_profiles WHERE tenant_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*models.ComplianceProfile
	for rows.Next() {
		profile := &models.ComplianceProfile{}
		var settingsJSON []byte

		if err := rows.Scan(
			&profile.ID, &profile.TenantID, &profile.Name, &profile.Framework,
			&profile.Description, &profile.Enabled, &settingsJSON,
			&profile.CreatedAt, &profile.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(settingsJSON, &profile.Settings)
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}

func (r *PostgresComplianceRepository) GetProfileByFramework(ctx context.Context, tenantID uuid.UUID, framework string) (*models.ComplianceProfile, error) {
	query := `
		SELECT id, tenant_id, name, framework, description, enabled, settings, created_at, updated_at
		FROM compliance_profiles WHERE tenant_id = $1 AND framework = $2
	`

	profile := &models.ComplianceProfile{}
	var settingsJSON []byte

	err := r.pool.QueryRow(ctx, query, tenantID, framework).Scan(
		&profile.ID, &profile.TenantID, &profile.Name, &profile.Framework,
		&profile.Description, &profile.Enabled, &settingsJSON,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(settingsJSON, &profile.Settings)
	return profile, nil
}

func (r *PostgresComplianceRepository) UpdateProfile(ctx context.Context, profile *models.ComplianceProfile) error {
	query := `
		UPDATE compliance_profiles
		SET name = $2, description = $3, enabled = $4, settings = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	settingsJSON, _ := json.Marshal(profile.Settings)
	_, err := r.pool.Exec(ctx, query, profile.ID, profile.Name, profile.Description, profile.Enabled, settingsJSON)
	return err
}

func (r *PostgresComplianceRepository) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM compliance_profiles WHERE id = $1", id)
	return err
}

// Retention policy operations

func (r *PostgresComplianceRepository) CreateRetentionPolicy(ctx context.Context, policy *models.DataRetentionPolicy) error {
	query := `
		INSERT INTO data_retention_policies (
			id, tenant_id, profile_id, name, description, data_category,
			retention_days, archive_enabled, archive_location, deletion_method,
			enabled, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	if policy.DeletionMethod == "" {
		policy.DeletionMethod = "soft"
	}

	_, err := r.pool.Exec(ctx, query,
		policy.ID, policy.TenantID, policy.ProfileID, policy.Name, policy.Description,
		policy.DataCategory, policy.RetentionDays, policy.ArchiveEnabled,
		policy.ArchiveLocation, policy.DeletionMethod, policy.Enabled,
	)

	return err
}

func (r *PostgresComplianceRepository) GetRetentionPolicy(ctx context.Context, id uuid.UUID) (*models.DataRetentionPolicy, error) {
	query := `
		SELECT id, tenant_id, profile_id, name, description, data_category,
		       retention_days, archive_enabled, archive_location, deletion_method,
		       enabled, last_execution, created_at, updated_at
		FROM data_retention_policies WHERE id = $1
	`

	policy := &models.DataRetentionPolicy{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&policy.ID, &policy.TenantID, &policy.ProfileID, &policy.Name, &policy.Description,
		&policy.DataCategory, &policy.RetentionDays, &policy.ArchiveEnabled,
		&policy.ArchiveLocation, &policy.DeletionMethod, &policy.Enabled,
		&policy.LastExecution, &policy.CreatedAt, &policy.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return policy, nil
}

func (r *PostgresComplianceRepository) GetRetentionPoliciesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.DataRetentionPolicy, error) {
	query := `
		SELECT id, tenant_id, profile_id, name, description, data_category,
		       retention_days, archive_enabled, archive_location, deletion_method,
		       enabled, last_execution, created_at, updated_at
		FROM data_retention_policies WHERE tenant_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*models.DataRetentionPolicy
	for rows.Next() {
		policy := &models.DataRetentionPolicy{}
		if err := rows.Scan(
			&policy.ID, &policy.TenantID, &policy.ProfileID, &policy.Name, &policy.Description,
			&policy.DataCategory, &policy.RetentionDays, &policy.ArchiveEnabled,
			&policy.ArchiveLocation, &policy.DeletionMethod, &policy.Enabled,
			&policy.LastExecution, &policy.CreatedAt, &policy.UpdatedAt,
		); err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return policies, nil
}

func (r *PostgresComplianceRepository) GetDueRetentionPolicies(ctx context.Context) ([]*models.DataRetentionPolicy, error) {
	query := `
		SELECT id, tenant_id, profile_id, name, description, data_category,
		       retention_days, archive_enabled, archive_location, deletion_method,
		       enabled, last_execution, created_at, updated_at
		FROM data_retention_policies
		WHERE enabled = true AND (
			last_execution IS NULL OR
			last_execution < CURRENT_TIMESTAMP - INTERVAL '1 day'
		)
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*models.DataRetentionPolicy
	for rows.Next() {
		policy := &models.DataRetentionPolicy{}
		if err := rows.Scan(
			&policy.ID, &policy.TenantID, &policy.ProfileID, &policy.Name, &policy.Description,
			&policy.DataCategory, &policy.RetentionDays, &policy.ArchiveEnabled,
			&policy.ArchiveLocation, &policy.DeletionMethod, &policy.Enabled,
			&policy.LastExecution, &policy.CreatedAt, &policy.UpdatedAt,
		); err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return policies, nil
}

func (r *PostgresComplianceRepository) UpdateRetentionPolicyExecution(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE data_retention_policies SET last_execution = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// PII pattern operations

func (r *PostgresComplianceRepository) CreatePIIPattern(ctx context.Context, pattern *models.PIIDetectionPattern) error {
	query := `
		INSERT INTO pii_detection_patterns (
			id, tenant_id, name, pattern_type, pattern_value, pii_category, sensitivity_level, enabled, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP
		)
	`

	if pattern.ID == uuid.Nil {
		pattern.ID = uuid.New()
	}
	if pattern.SensitivityLevel == "" {
		pattern.SensitivityLevel = models.SensitivityMedium
	}

	_, err := r.pool.Exec(ctx, query,
		pattern.ID, pattern.TenantID, pattern.Name, pattern.PatternType,
		pattern.PatternValue, pattern.PIICategory, pattern.SensitivityLevel, pattern.Enabled,
	)

	return err
}

func (r *PostgresComplianceRepository) GetPIIPatterns(ctx context.Context, tenantID *uuid.UUID) ([]*models.PIIDetectionPattern, error) {
	query := `
		SELECT id, tenant_id, name, pattern_type, pattern_value, pii_category, sensitivity_level, enabled, created_at
		FROM pii_detection_patterns
		WHERE enabled = true AND (tenant_id IS NULL OR tenant_id = $1)
		ORDER BY sensitivity_level DESC, name ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []*models.PIIDetectionPattern
	for rows.Next() {
		pattern := &models.PIIDetectionPattern{}
		if err := rows.Scan(
			&pattern.ID, &pattern.TenantID, &pattern.Name, &pattern.PatternType,
			&pattern.PatternValue, &pattern.PIICategory, &pattern.SensitivityLevel,
			&pattern.Enabled, &pattern.CreatedAt,
		); err != nil {
			return nil, err
		}
		patterns = append(patterns, pattern)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

func (r *PostgresComplianceRepository) DeletePIIPattern(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM pii_detection_patterns WHERE id = $1", id)
	return err
}

// PII detection operations

func (r *PostgresComplianceRepository) CreatePIIDetection(ctx context.Context, detection *models.PIIDetection) error {
	query := `
		INSERT INTO pii_detections (
			id, tenant_id, pattern_id, source_type, source_id, field_path,
			pii_category, sensitivity_level, redaction_applied, detected_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP
		)
	`

	if detection.ID == uuid.Nil {
		detection.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx, query,
		detection.ID, detection.TenantID, detection.PatternID, detection.SourceType,
		detection.SourceID, detection.FieldPath, detection.PIICategory,
		detection.SensitivityLevel, detection.RedactionApplied,
	)

	return err
}

func (r *PostgresComplianceRepository) GetPIIDetectionsBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) ([]*models.PIIDetection, error) {
	query := `
		SELECT id, tenant_id, pattern_id, source_type, source_id, field_path,
		       pii_category, sensitivity_level, redaction_applied, detected_at
		FROM pii_detections WHERE source_type = $1 AND source_id = $2
		ORDER BY detected_at DESC
	`

	return r.queryDetections(ctx, query, sourceType, sourceID)
}

func (r *PostgresComplianceRepository) GetPIIDetectionsByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.PIIDetection, error) {
	query := `
		SELECT id, tenant_id, pattern_id, source_type, source_id, field_path,
		       pii_category, sensitivity_level, redaction_applied, detected_at
		FROM pii_detections WHERE tenant_id = $1
		ORDER BY detected_at DESC LIMIT $2
	`

	return r.queryDetections(ctx, query, tenantID, limit)
}

func (r *PostgresComplianceRepository) CountPIIDetectionsToday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM pii_detections
		WHERE tenant_id = $1 AND detected_at >= CURRENT_DATE
	`

	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	return count, err
}

func (r *PostgresComplianceRepository) queryDetections(ctx context.Context, query string, args ...interface{}) ([]*models.PIIDetection, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var detections []*models.PIIDetection
	for rows.Next() {
		d := &models.PIIDetection{}
		if err := rows.Scan(
			&d.ID, &d.TenantID, &d.PatternID, &d.SourceType, &d.SourceID,
			&d.FieldPath, &d.PIICategory, &d.SensitivityLevel,
			&d.RedactionApplied, &d.DetectedAt,
		); err != nil {
			return nil, err
		}
		detections = append(detections, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return detections, nil
}

// Audit log operations

func (r *PostgresComplianceRepository) CreateAuditLog(ctx context.Context, log *models.ComplianceAuditLog) error {
	query := `
		INSERT INTO compliance_audit_logs (
			id, tenant_id, actor_id, actor_type, action, resource_type, resource_id,
			details, ip_address, user_agent, timestamp, retention_until
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP, $11
		)
	`

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	detailsJSON, _ := json.Marshal(log.Details)

	_, err := r.pool.Exec(ctx, query,
		log.ID, log.TenantID, log.ActorID, log.ActorType, log.Action,
		log.ResourceType, log.ResourceID, detailsJSON, log.IPAddress,
		log.UserAgent, log.RetentionUntil,
	)

	return err
}

func (r *PostgresComplianceRepository) QueryAuditLogs(ctx context.Context, query *models.AuditLogQuery) ([]*models.ComplianceAuditLog, error) {
	sql := `
		SELECT id, tenant_id, actor_id, actor_type, action, resource_type, resource_id,
		       details, ip_address, user_agent, timestamp, retention_until
		FROM compliance_audit_logs
		WHERE tenant_id = $1
	`

	args := []interface{}{query.TenantID}
	argNum := 2

	if query.ActorID != nil {
		sql += fmt.Sprintf(" AND actor_id = $%d", argNum)
		args = append(args, *query.ActorID)
		argNum++
	}
	if query.Action != "" {
		sql += fmt.Sprintf(" AND action = $%d", argNum)
		args = append(args, query.Action)
		argNum++
	}
	if query.ResourceType != "" {
		sql += fmt.Sprintf(" AND resource_type = $%d", argNum)
		args = append(args, query.ResourceType)
		argNum++
	}
	if query.StartTime != nil {
		sql += fmt.Sprintf(" AND timestamp >= $%d", argNum)
		args = append(args, *query.StartTime)
		argNum++
	}
	if query.EndTime != nil {
		sql += fmt.Sprintf(" AND timestamp <= $%d", argNum)
		args = append(args, *query.EndTime)
		argNum++
	}

	sql += " ORDER BY timestamp DESC"

	if query.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, query.Limit)
		argNum++
	}
	if query.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, query.Offset)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.ComplianceAuditLog
	for rows.Next() {
		log := &models.ComplianceAuditLog{}
		var detailsJSON []byte

		if err := rows.Scan(
			&log.ID, &log.TenantID, &log.ActorID, &log.ActorType, &log.Action,
			&log.ResourceType, &log.ResourceID, &detailsJSON, &log.IPAddress,
			&log.UserAgent, &log.Timestamp, &log.RetentionUntil,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(detailsJSON, &log.Details)
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *PostgresComplianceRepository) GetAuditLogsByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*models.ComplianceAuditLog, error) {
	query := &models.AuditLogQuery{
		ResourceType: resourceType,
		Limit:        100,
	}
	// Need tenant ID, but this is a simplified version
	return r.QueryAuditLogs(ctx, query)
}

// Report operations

func (r *PostgresComplianceRepository) CreateReport(ctx context.Context, report *models.ComplianceReport) error {
	query := `
		INSERT INTO compliance_reports (
			id, tenant_id, profile_id, report_type, title, description, status,
			period_start, period_end, report_data, generated_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, CURRENT_TIMESTAMP
		)
	`

	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}
	if report.Status == "" {
		report.Status = models.ReportStatusPending
	}

	reportDataJSON, _ := json.Marshal(report.ReportData)

	_, err := r.pool.Exec(ctx, query,
		report.ID, report.TenantID, report.ProfileID, report.ReportType,
		report.Title, report.Description, report.Status, report.PeriodStart,
		report.PeriodEnd, reportDataJSON, report.GeneratedBy,
	)

	return err
}

func (r *PostgresComplianceRepository) GetReport(ctx context.Context, id uuid.UUID) (*models.ComplianceReport, error) {
	query := `
		SELECT id, tenant_id, profile_id, report_type, title, description, status,
		       period_start, period_end, report_data, artifact_url, generated_by,
		       created_at, completed_at
		FROM compliance_reports WHERE id = $1
	`

	report := &models.ComplianceReport{}
	var reportDataJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&report.ID, &report.TenantID, &report.ProfileID, &report.ReportType,
		&report.Title, &report.Description, &report.Status, &report.PeriodStart,
		&report.PeriodEnd, &reportDataJSON, &report.ArtifactURL, &report.GeneratedBy,
		&report.CreatedAt, &report.CompletedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(reportDataJSON, &report.ReportData)
	return report, nil
}

func (r *PostgresComplianceRepository) GetReportsByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.ComplianceReport, error) {
	query := `
		SELECT id, tenant_id, profile_id, report_type, title, description, status,
		       period_start, period_end, report_data, artifact_url, generated_by,
		       created_at, completed_at
		FROM compliance_reports WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*models.ComplianceReport
	for rows.Next() {
		report := &models.ComplianceReport{}
		var reportDataJSON []byte

		if err := rows.Scan(
			&report.ID, &report.TenantID, &report.ProfileID, &report.ReportType,
			&report.Title, &report.Description, &report.Status, &report.PeriodStart,
			&report.PeriodEnd, &reportDataJSON, &report.ArtifactURL, &report.GeneratedBy,
			&report.CreatedAt, &report.CompletedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(reportDataJSON, &report.ReportData)
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}

func (r *PostgresComplianceRepository) UpdateReportStatus(ctx context.Context, id uuid.UUID, status string, reportData map[string]interface{}, artifactURL string) error {
	query := `
		UPDATE compliance_reports
		SET status = $2, report_data = $3, artifact_url = $4, completed_at = $5
		WHERE id = $1
	`

	reportDataJSON, _ := json.Marshal(reportData)
	var completedAt *time.Time
	if status == models.ReportStatusCompleted || status == models.ReportStatusFailed {
		now := time.Now()
		completedAt = &now
	}

	_, err := r.pool.Exec(ctx, query, id, status, reportDataJSON, artifactURL, completedAt)
	return err
}

// Finding operations

func (r *PostgresComplianceRepository) CreateFinding(ctx context.Context, finding *models.ComplianceFinding) error {
	query := `
		INSERT INTO compliance_findings (
			id, report_id, tenant_id, severity, category, title, description,
			recommendation, status, remediation_deadline, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP
		)
	`

	if finding.ID == uuid.Nil {
		finding.ID = uuid.New()
	}
	if finding.Status == "" {
		finding.Status = models.FindingStatusOpen
	}

	_, err := r.pool.Exec(ctx, query,
		finding.ID, finding.ReportID, finding.TenantID, finding.Severity,
		finding.Category, finding.Title, finding.Description, finding.Recommendation,
		finding.Status, finding.RemediationDeadline,
	)

	return err
}

func (r *PostgresComplianceRepository) GetFindingsByReport(ctx context.Context, reportID uuid.UUID) ([]*models.ComplianceFinding, error) {
	query := `
		SELECT id, report_id, tenant_id, severity, category, title, description,
		       recommendation, status, remediation_deadline, remediated_at, created_at
		FROM compliance_findings WHERE report_id = $1
		ORDER BY CASE severity
			WHEN 'critical' THEN 1
			WHEN 'high' THEN 2
			WHEN 'medium' THEN 3
			WHEN 'low' THEN 4
			ELSE 5
		END, created_at DESC
	`

	return r.queryFindings(ctx, query, reportID)
}

func (r *PostgresComplianceRepository) GetOpenFindingsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ComplianceFinding, error) {
	query := `
		SELECT id, report_id, tenant_id, severity, category, title, description,
		       recommendation, status, remediation_deadline, remediated_at, created_at
		FROM compliance_findings WHERE tenant_id = $1 AND status IN ('open', 'acknowledged')
		ORDER BY CASE severity
			WHEN 'critical' THEN 1
			WHEN 'high' THEN 2
			WHEN 'medium' THEN 3
			WHEN 'low' THEN 4
			ELSE 5
		END, created_at DESC
	`

	return r.queryFindings(ctx, query, tenantID)
}

func (r *PostgresComplianceRepository) queryFindings(ctx context.Context, query string, arg interface{}) ([]*models.ComplianceFinding, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []*models.ComplianceFinding
	for rows.Next() {
		f := &models.ComplianceFinding{}
		if err := rows.Scan(
			&f.ID, &f.ReportID, &f.TenantID, &f.Severity, &f.Category,
			&f.Title, &f.Description, &f.Recommendation, &f.Status,
			&f.RemediationDeadline, &f.RemediatedAt, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		findings = append(findings, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return findings, nil
}

func (r *PostgresComplianceRepository) UpdateFindingStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE compliance_findings SET status = $2, remediated_at = $3 WHERE id = $1`

	var remediatedAt *time.Time
	if status == models.FindingStatusRemediated {
		now := time.Now()
		remediatedAt = &now
	}

	_, err := r.pool.Exec(ctx, query, id, status, remediatedAt)
	return err
}

func (r *PostgresComplianceRepository) CountFindingsBySeverity(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT severity, COUNT(*) FROM compliance_findings
		WHERE tenant_id = $1 AND status IN ('open', 'acknowledged')
		GROUP BY severity
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, err
		}
		counts[severity] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return counts, nil
}

// DSR operations

func (r *PostgresComplianceRepository) CreateDSR(ctx context.Context, dsr *models.DataSubjectRequest) error {
	query := `
		INSERT INTO data_subject_requests (
			id, tenant_id, request_type, data_subject_id, data_subject_email,
			status, request_details, deadline, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	if dsr.ID == uuid.Nil {
		dsr.ID = uuid.New()
	}
	if dsr.Status == "" {
		dsr.Status = models.DSRStatusPending
	}
	// GDPR requires response within 30 days
	if dsr.Deadline == nil {
		deadline := time.Now().AddDate(0, 0, 30)
		dsr.Deadline = &deadline
	}

	requestDetailsJSON, _ := json.Marshal(dsr.RequestDetails)

	_, err := r.pool.Exec(ctx, query,
		dsr.ID, dsr.TenantID, dsr.RequestType, dsr.DataSubjectID,
		dsr.DataSubjectEmail, dsr.Status, requestDetailsJSON, dsr.Deadline,
	)

	return err
}

func (r *PostgresComplianceRepository) GetDSR(ctx context.Context, id uuid.UUID) (*models.DataSubjectRequest, error) {
	query := `
		SELECT id, tenant_id, request_type, data_subject_id, data_subject_email,
		       status, request_details, response_data, deadline, completed_at,
		       created_at, updated_at
		FROM data_subject_requests WHERE id = $1
	`

	dsr := &models.DataSubjectRequest{}
	var requestDetailsJSON, responseDataJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&dsr.ID, &dsr.TenantID, &dsr.RequestType, &dsr.DataSubjectID,
		&dsr.DataSubjectEmail, &dsr.Status, &requestDetailsJSON, &responseDataJSON,
		&dsr.Deadline, &dsr.CompletedAt, &dsr.CreatedAt, &dsr.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(requestDetailsJSON, &dsr.RequestDetails)
	json.Unmarshal(responseDataJSON, &dsr.ResponseData)

	return dsr, nil
}

func (r *PostgresComplianceRepository) GetDSRsByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]*models.DataSubjectRequest, error) {
	query := `
		SELECT id, tenant_id, request_type, data_subject_id, data_subject_email,
		       status, request_details, response_data, deadline, completed_at,
		       created_at, updated_at
		FROM data_subject_requests WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dsrs []*models.DataSubjectRequest
	for rows.Next() {
		dsr := &models.DataSubjectRequest{}
		var requestDetailsJSON, responseDataJSON []byte

		if err := rows.Scan(
			&dsr.ID, &dsr.TenantID, &dsr.RequestType, &dsr.DataSubjectID,
			&dsr.DataSubjectEmail, &dsr.Status, &requestDetailsJSON, &responseDataJSON,
			&dsr.Deadline, &dsr.CompletedAt, &dsr.CreatedAt, &dsr.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(requestDetailsJSON, &dsr.RequestDetails)
		json.Unmarshal(responseDataJSON, &dsr.ResponseData)
		dsrs = append(dsrs, dsr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dsrs, nil
}

func (r *PostgresComplianceRepository) UpdateDSRStatus(ctx context.Context, id uuid.UUID, status string, responseData map[string]interface{}) error {
	query := `
		UPDATE data_subject_requests
		SET status = $2, response_data = $3, completed_at = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	responseDataJSON, _ := json.Marshal(responseData)
	var completedAt *time.Time
	if status == models.DSRStatusCompleted || status == models.DSRStatusRejected {
		now := time.Now()
		completedAt = &now
	}

	_, err := r.pool.Exec(ctx, query, id, status, responseDataJSON, completedAt)
	return err
}

func (r *PostgresComplianceRepository) CountPendingDSRs(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM data_subject_requests WHERE tenant_id = $1 AND status IN ('pending', 'processing')`
	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	return count, err
}
