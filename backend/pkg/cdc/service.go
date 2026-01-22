package cdc

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Service provides CDC functionality
type Service struct {
	repo        Repository
	secrets     SecretStore
	connectors  sync.Map // map[connectorID]*ConnectorRunner
	config      *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxConnectorsPerTenant int
	HealthCheckInterval    time.Duration
	BatchFlushInterval     time.Duration
	DefaultBatchSize       int
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxConnectorsPerTenant: 10,
		HealthCheckInterval:    30 * time.Second,
		BatchFlushInterval:     5 * time.Second,
		DefaultBatchSize:       100,
	}
}

// NewService creates a new CDC service
func NewService(repo Repository, secrets SecretStore, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	return &Service{
		repo:    repo,
		secrets: secrets,
		config:  config,
	}
}

// CreateConnector creates a new CDC connector
func (s *Service) CreateConnector(ctx context.Context, tenantID string, req *CreateConnectorRequest) (*CDCConnector, error) {
	// Check connector limit
	existing, err := s.repo.ListConnectors(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}
	if len(existing) >= s.config.MaxConnectorsPerTenant {
		return nil, fmt.Errorf("maximum connectors reached: %d", s.config.MaxConnectorsPerTenant)
	}

	conn := &CDCConnector{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		Name:             req.Name,
		Type:             req.Type,
		Status:           StatusCreated,
		ConnectionConfig: req.ConnectionConfig,
		CaptureConfig:    req.CaptureConfig,
		WebhookConfig:    req.WebhookConfig,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if req.OffsetConfig != nil {
		conn.OffsetConfig = *req.OffsetConfig
	} else {
		conn.OffsetConfig = OffsetConfig{
			StorageType:     "database",
			FlushIntervalMs: 1000,
			CommitOnSuccess: true,
		}
	}

	// Set defaults for capture config
	if len(conn.CaptureConfig.Operations) == 0 {
		conn.CaptureConfig.Operations = []ChangeOperation{OpInsert, OpUpdate, OpDelete}
	}
	if conn.CaptureConfig.SnapshotMode == "" {
		conn.CaptureConfig.SnapshotMode = "never"
	}

	// Store password securely
	if req.ConnectionConfig.Password != "" {
		if err := s.secrets.StoreConnectionCredentials(ctx, conn.ID, req.ConnectionConfig.Password); err != nil {
			return nil, fmt.Errorf("failed to store credentials: %w", err)
		}
		conn.ConnectionConfig.Password = "" // Clear from memory
	}

	if err := s.repo.SaveConnector(ctx, conn); err != nil {
		return nil, err
	}

	return conn, nil
}

// GetConnector retrieves a connector
func (s *Service) GetConnector(ctx context.Context, tenantID, connID string) (*CDCConnector, error) {
	return s.repo.GetConnector(ctx, tenantID, connID)
}

// ListConnectors lists connectors
func (s *Service) ListConnectors(ctx context.Context, tenantID string, status *ConnectorStatus) ([]CDCConnector, error) {
	return s.repo.ListConnectors(ctx, tenantID, status)
}

// UpdateConnector updates a connector
func (s *Service) UpdateConnector(ctx context.Context, tenantID, connID string, req *UpdateConnectorRequest) (*CDCConnector, error) {
	conn, err := s.repo.GetConnector(ctx, tenantID, connID)
	if err != nil {
		return nil, err
	}

	// Cannot update running connector
	if conn.Status == StatusRunning {
		return nil, fmt.Errorf("cannot update running connector, stop it first")
	}

	if req.Name != nil {
		conn.Name = *req.Name
	}
	if req.ConnectionConfig != nil {
		conn.ConnectionConfig = *req.ConnectionConfig
		// Store new password if provided
		if req.ConnectionConfig.Password != "" {
			s.secrets.StoreConnectionCredentials(ctx, conn.ID, req.ConnectionConfig.Password)
			conn.ConnectionConfig.Password = ""
		}
	}
	if req.CaptureConfig != nil {
		conn.CaptureConfig = *req.CaptureConfig
	}
	if req.WebhookConfig != nil {
		conn.WebhookConfig = *req.WebhookConfig
	}

	conn.UpdatedAt = time.Now()

	if err := s.repo.SaveConnector(ctx, conn); err != nil {
		return nil, err
	}

	return conn, nil
}

// DeleteConnector deletes a connector
func (s *Service) DeleteConnector(ctx context.Context, tenantID, connID string) error {
	conn, err := s.repo.GetConnector(ctx, tenantID, connID)
	if err != nil {
		return err
	}

	// Stop if running
	if conn.Status == StatusRunning {
		if _, err := s.StopConnector(ctx, tenantID, connID); err != nil {
			return fmt.Errorf("failed to stop connector: %w", err)
		}
	}

	// Delete credentials
	s.secrets.DeleteConnectionCredentials(ctx, connID)

	return s.repo.DeleteConnector(ctx, tenantID, connID)
}

// StartConnector starts a CDC connector
func (s *Service) StartConnector(ctx context.Context, tenantID, connID string) (*CDCConnector, error) {
	conn, err := s.repo.GetConnector(ctx, tenantID, connID)
	if err != nil {
		return nil, err
	}

	if conn.Status == StatusRunning {
		return conn, nil // Already running
	}

	// Get password
	password, err := s.secrets.GetConnectionCredentials(ctx, connID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}
	conn.ConnectionConfig.Password = password

	// Create and start runner
	runner := NewConnectorRunner(conn, s.repo, s.config)
	s.connectors.Store(connID, runner)

	if err := runner.Start(ctx); err != nil {
		s.connectors.Delete(connID)
		s.repo.UpdateConnectorStatus(ctx, tenantID, connID, StatusError, err.Error())
		return nil, err
	}

	s.repo.UpdateConnectorStatus(ctx, tenantID, connID, StatusRunning, "")

	conn.Status = StatusRunning
	conn.ErrorMessage = ""
	return conn, nil
}

// StopConnector stops a CDC connector
func (s *Service) StopConnector(ctx context.Context, tenantID, connID string) (*CDCConnector, error) {
	conn, err := s.repo.GetConnector(ctx, tenantID, connID)
	if err != nil {
		return nil, err
	}

	if runnerI, ok := s.connectors.Load(connID); ok {
		runner := runnerI.(*ConnectorRunner)
		runner.Stop()
		s.connectors.Delete(connID)
	}

	s.repo.UpdateConnectorStatus(ctx, tenantID, connID, StatusStopped, "")

	conn.Status = StatusStopped
	return conn, nil
}

// PauseConnector pauses a CDC connector
func (s *Service) PauseConnector(ctx context.Context, tenantID, connID string) (*CDCConnector, error) {
	conn, err := s.repo.GetConnector(ctx, tenantID, connID)
	if err != nil {
		return nil, err
	}

	if runnerI, ok := s.connectors.Load(connID); ok {
		runner := runnerI.(*ConnectorRunner)
		runner.Pause()
	}

	s.repo.UpdateConnectorStatus(ctx, tenantID, connID, StatusPaused, "")

	conn.Status = StatusPaused
	return conn, nil
}

// TestConnection tests database connectivity
func (s *Service) TestConnection(ctx context.Context, tenantID string, req *TestConnectionRequest) (*TestConnectionResult, error) {
	result := &TestConnectionResult{}

	dsn := buildDSN(req.Type, &req.ConnectionConfig)

	driverName := getDriverName(req.Type)
	if driverName == "" {
		result.Message = "Unsupported connector type"
		return result, nil
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to connect: %v", err)
		return result, nil
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		result.Message = fmt.Sprintf("Connection failed: %v", err)
		return result, nil
	}

	result.Success = true
	result.Message = "Connection successful"

	// Get version
	var version string
	switch req.Type {
	case ConnectorPostgreSQL:
		db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
		result.ReplicationOK = checkPostgresReplication(ctx, db, &req.ConnectionConfig)
	case ConnectorMySQL:
		db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
		result.ReplicationOK = checkMySQLBinlog(ctx, db)
	}
	result.Version = version

	// List available tables
	result.AvailableTables = listTables(ctx, db, req.Type)

	// Add warnings
	if !result.ReplicationOK {
		result.Warnings = append(result.Warnings,
			"Replication/CDC not configured. Please enable logical replication.")
	}

	return result, nil
}

// GetSchema retrieves table schema information
func (s *Service) GetSchema(ctx context.Context, tenantID, connID string, schema, table string) (*SchemaInfo, error) {
	conn, err := s.repo.GetConnector(ctx, tenantID, connID)
	if err != nil {
		return nil, err
	}

	password, _ := s.secrets.GetConnectionCredentials(ctx, connID)
	conn.ConnectionConfig.Password = password

	dsn := buildDSN(conn.Type, &conn.ConnectionConfig)
	db, err := sql.Open(getDriverName(conn.Type), dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	return getTableSchema(ctx, db, conn.Type, schema, table)
}

// GetMetrics retrieves connector metrics
func (s *Service) GetMetrics(ctx context.Context, tenantID, connID string) (*CDCMetrics, error) {
	return s.repo.GetConnectorMetrics(ctx, tenantID, connID)
}

// GetHealth retrieves connector health
func (s *Service) GetHealth(ctx context.Context, tenantID, connID string) (*ConnectorHealth, error) {
	conn, err := s.repo.GetConnector(ctx, tenantID, connID)
	if err != nil {
		return nil, err
	}

	health := &ConnectorHealth{
		ConnectorID: connID,
		LastCheck:   time.Now(),
	}

	// Check if running
	if runnerI, ok := s.connectors.Load(connID); ok {
		runner := runnerI.(*ConnectorRunner)
		health.Healthy = runner.IsHealthy()
		health.ConnectionOK = runner.IsConnected()
		health.ReplicationOK = runner.IsReplicating()
		health.LagAcceptable = runner.GetLag() < 60*time.Second
		health.ErrorCount = runner.GetErrorCount()
	} else {
		health.Healthy = conn.Status != StatusError
		health.ConnectionOK = conn.Status == StatusRunning || conn.Status == StatusPaused
	}

	// Add warnings
	if conn.LastEventAt != nil {
		lag := time.Since(*conn.LastEventAt)
		if lag > 5*time.Minute {
			health.Warnings = append(health.Warnings,
				fmt.Sprintf("High replication lag: %v", lag))
		}
	}

	if conn.EventsFailed > 0 && conn.EventsProcessed > 0 {
		errorRate := float64(conn.EventsFailed) / float64(conn.EventsProcessed)
		if errorRate > 0.01 {
			health.Warnings = append(health.Warnings,
				fmt.Sprintf("High error rate: %.2f%%", errorRate*100))
		}
	}

	return health, nil
}

// GetEventHistory retrieves event history
func (s *Service) GetEventHistory(ctx context.Context, tenantID, connID string, limit, offset int) ([]EventHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	return s.repo.GetEventHistory(ctx, tenantID, connID, limit, offset)
}

// GetSupportedConnectors returns supported connector types
func (s *Service) GetSupportedConnectors() []ConnectorTypeInfo {
	return GetSupportedConnectors()
}

// ConnectorRunner handles the actual CDC streaming
type ConnectorRunner struct {
	conn      *CDCConnector
	repo      Repository
	config    *ServiceConfig
	stopCh    chan struct{}
	pauseCh   chan struct{}
	resumeCh  chan struct{}
	mu        sync.Mutex
	running   bool
	paused    bool
	healthy   bool
	connected bool
	replicating bool
	lag       time.Duration
	errorCount int
}

// NewConnectorRunner creates a new connector runner
func NewConnectorRunner(conn *CDCConnector, repo Repository, config *ServiceConfig) *ConnectorRunner {
	return &ConnectorRunner{
		conn:     conn,
		repo:     repo,
		config:   config,
		stopCh:   make(chan struct{}),
		pauseCh:  make(chan struct{}),
		resumeCh: make(chan struct{}),
	}
}

// Start starts the connector
func (r *ConnectorRunner) Start(ctx context.Context) error {
	// Test connection first
	dsn := buildDSN(r.conn.Type, &r.conn.ConnectionConfig)
	db, err := sql.Open(getDriverName(r.conn.Type), dsn)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("ping failed: %w", err)
	}

	r.mu.Lock()
	r.running = true
	r.connected = true
	r.healthy = true
	r.mu.Unlock()

	// Start streaming in background
	go r.stream(db)

	return nil
}

// Stop stops the connector
func (r *ConnectorRunner) Stop() {
	close(r.stopCh)
	r.mu.Lock()
	r.running = false
	r.mu.Unlock()
}

// Pause pauses the connector
func (r *ConnectorRunner) Pause() {
	r.mu.Lock()
	r.paused = true
	r.mu.Unlock()
}

// Resume resumes the connector
func (r *ConnectorRunner) Resume() {
	r.mu.Lock()
	r.paused = false
	r.mu.Unlock()
	select {
	case r.resumeCh <- struct{}{}:
	default:
	}
}

func (r *ConnectorRunner) IsHealthy() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.healthy
}

func (r *ConnectorRunner) IsConnected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connected
}

func (r *ConnectorRunner) IsReplicating() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.replicating
}

func (r *ConnectorRunner) GetLag() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lag
}

func (r *ConnectorRunner) GetErrorCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.errorCount
}

func (r *ConnectorRunner) stream(db *sql.DB) {
	defer db.Close()

	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.mu.Lock()
			if r.paused {
				r.mu.Unlock()
				continue
			}
			r.mu.Unlock()

			// In production, this would implement actual WAL/binlog streaming
			// For now, simulate periodic polling
			r.pollChanges(context.Background(), db)
		}
	}
}

func (r *ConnectorRunner) pollChanges(ctx context.Context, db *sql.DB) {
	r.mu.Lock()
	r.replicating = true
	r.mu.Unlock()

	// This is a simplified implementation
	// Production would use pg_logical_slot_get_changes or binlog streaming
	
	for _, table := range r.conn.CaptureConfig.Tables {
		// Check for changes (simplified)
		_ = table
	}

	r.mu.Lock()
	r.lag = 0 // Would calculate actual lag
	r.mu.Unlock()
}

// Helper functions

func buildDSN(connType ConnectorType, cfg *ConnectionConfig) string {
	switch connType {
	case ConnectorPostgreSQL:
		sslMode := cfg.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, sslMode)
	case ConnectorMySQL:
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	default:
		return ""
	}
}

func getDriverName(connType ConnectorType) string {
	switch connType {
	case ConnectorPostgreSQL:
		return "pgx"
	case ConnectorMySQL:
		return "mysql"
	default:
		return ""
	}
}

func checkPostgresReplication(ctx context.Context, db *sql.DB, cfg *ConnectionConfig) bool {
	var walLevel string
	err := db.QueryRowContext(ctx, "SHOW wal_level").Scan(&walLevel)
	if err != nil {
		return false
	}
	return walLevel == "logical"
}

func checkMySQLBinlog(ctx context.Context, db *sql.DB) bool {
	var logBin string
	err := db.QueryRowContext(ctx, "SELECT @@log_bin").Scan(&logBin)
	if err != nil {
		return false
	}
	return logBin == "1" || strings.ToUpper(logBin) == "ON"
}

func listTables(ctx context.Context, db *sql.DB, connType ConnectorType) []string {
	var query string
	switch connType {
	case ConnectorPostgreSQL:
		query = `SELECT table_schema || '.' || table_name FROM information_schema.tables
				 WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
				 ORDER BY table_schema, table_name LIMIT 100`
	case ConnectorMySQL:
		query = `SELECT CONCAT(TABLE_SCHEMA, '.', TABLE_NAME) FROM information_schema.TABLES
				 WHERE TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
				 ORDER BY TABLE_SCHEMA, TABLE_NAME LIMIT 100`
	default:
		return nil
	}

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if rows.Scan(&table) == nil {
			tables = append(tables, table)
		}
	}
	return tables
}

func getTableSchema(ctx context.Context, db *sql.DB, connType ConnectorType, schema, table string) (*SchemaInfo, error) {
	info := &SchemaInfo{
		Schema: schema,
		Table:  table,
	}

	var query string
	switch connType {
	case ConnectorPostgreSQL:
		query = `
			SELECT column_name, data_type, is_nullable,
				   CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_pk,
				   column_default
			FROM information_schema.columns c
			LEFT JOIN (
				SELECT ku.column_name
				FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage ku
					ON tc.constraint_name = ku.constraint_name
				WHERE tc.constraint_type = 'PRIMARY KEY'
					AND tc.table_schema = $1 AND tc.table_name = $2
			) pk ON pk.column_name = c.column_name
			WHERE c.table_schema = $1 AND c.table_name = $2
			ORDER BY c.ordinal_position`
	case ConnectorMySQL:
		query = `
			SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE,
				   COLUMN_KEY = 'PRI' as is_pk,
				   COLUMN_DEFAULT
			FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION`
	default:
		return nil, fmt.Errorf("unsupported connector type")
	}

	rows, err := db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var nullable, dflt sql.NullString

		err := rows.Scan(&col.Name, &col.Type, &nullable, &col.PrimaryKey, &dflt)
		if err != nil {
			continue
		}

		col.Nullable = nullable.String == "YES"
		if dflt.Valid {
			col.Default = dflt.String
		}

		info.Columns = append(info.Columns, col)
		if col.PrimaryKey {
			info.KeyColumns = append(info.KeyColumns, col.Name)
		}
	}

	return info, nil
}
