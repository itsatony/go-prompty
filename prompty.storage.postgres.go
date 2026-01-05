package prompty

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresConfig configures the PostgreSQL storage driver.
type PostgresConfig struct {
	// ConnectionString is the PostgreSQL connection DSN.
	// Format: "postgres://user:password@host:port/database?sslmode=disable"
	ConnectionString string

	// MaxOpenConns is the maximum number of open connections.
	// Default: 25
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	// Default: 5
	MaxIdleConns int

	// ConnMaxLifetime is the maximum connection lifetime.
	// Default: 5 minutes
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum idle time for connections.
	// Default: 5 minutes
	ConnMaxIdleTime time.Duration

	// TablePrefix allows customizing the table name prefix.
	// Default: "prompty_"
	TablePrefix string

	// AutoMigrate runs schema migrations on Open.
	// Default: false
	AutoMigrate bool

	// QueryTimeout is the default timeout for queries.
	// Default: 30 seconds
	QueryTimeout time.Duration
}

// DefaultPostgresConfig returns a configuration with sensible defaults.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		MaxOpenConns:    PostgresDefaultMaxOpenConns,
		MaxIdleConns:    PostgresDefaultMaxIdleConns,
		ConnMaxLifetime: PostgresDefaultConnMaxLifetime,
		ConnMaxIdleTime: PostgresDefaultConnMaxIdleTime,
		TablePrefix:     PostgresTablePrefix,
		AutoMigrate:     false,
		QueryTimeout:    PostgresDefaultQueryTimeout,
	}
}

// PostgresStorage implements TemplateStorage using PostgreSQL.
type PostgresStorage struct {
	db     *sql.DB
	config PostgresConfig
	mu     sync.RWMutex
	closed bool
}

// PostgresStorageDriver is the driver for creating PostgresStorage instances.
type PostgresStorageDriver struct{}

func init() {
	RegisterStorageDriver(StorageDriverNamePostgres, &PostgresStorageDriver{})
}

// Open creates a new PostgresStorage instance.
// The connection string should be a PostgreSQL DSN.
func (d *PostgresStorageDriver) Open(connectionString string) (TemplateStorage, error) {
	config := DefaultPostgresConfig()
	config.ConnectionString = connectionString
	config.AutoMigrate = true // Auto-migrate when opened via driver registry
	return NewPostgresStorage(config)
}

// NewPostgresStorage creates a new PostgreSQL template storage.
func NewPostgresStorage(config PostgresConfig) (*PostgresStorage, error) {
	if config.ConnectionString == "" {
		return nil, &StorageError{Message: ErrMsgPostgresEmptyConnString}
	}

	// Apply defaults for zero values
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = PostgresDefaultMaxOpenConns
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = PostgresDefaultMaxIdleConns
	}
	if config.ConnMaxLifetime == 0 {
		config.ConnMaxLifetime = PostgresDefaultConnMaxLifetime
	}
	if config.ConnMaxIdleTime == 0 {
		config.ConnMaxIdleTime = PostgresDefaultConnMaxIdleTime
	}
	if config.TablePrefix == "" {
		config.TablePrefix = PostgresTablePrefix
	}
	if config.QueryTimeout == 0 {
		config.QueryTimeout = PostgresDefaultQueryTimeout
	}

	db, err := sql.Open("postgres", config.ConnectionString)
	if err != nil {
		return nil, &StorageError{
			Message: ErrMsgPostgresConnectionFailed,
			Cause:   err,
		}
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), config.QueryTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, &StorageError{
			Message: ErrMsgPostgresConnectionFailed,
			Cause:   err,
		}
	}

	storage := &PostgresStorage{
		db:     db,
		config: config,
	}

	// Run migrations if configured
	if config.AutoMigrate {
		if err := storage.RunMigrations(ctx); err != nil {
			db.Close()
			return nil, err
		}
	}

	return storage, nil
}

// MustNewPostgresStorage creates a new PostgreSQL storage or panics.
func MustNewPostgresStorage(config PostgresConfig) *PostgresStorage {
	storage, err := NewPostgresStorage(config)
	if err != nil {
		panic(err)
	}
	return storage
}

// tableName returns the full table name with prefix.
func (s *PostgresStorage) tableName() string {
	return s.config.TablePrefix + "templates"
}

// migrationsTableName returns the migrations table name with prefix.
func (s *PostgresStorage) migrationsTableName() string {
	return s.config.TablePrefix + "schema_migrations"
}

// Get retrieves the latest version of a template by name.
func (s *PostgresStorage) Get(ctx context.Context, name string) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	query := fmt.Sprintf(`
		SELECT id, name, source, version, metadata, inference_config,
		       created_at, updated_at, created_by, tenant_id, tags
		FROM %s
		WHERE name = $1
		ORDER BY version DESC
		LIMIT 1`, s.tableName())

	row := s.db.QueryRowContext(ctx, query, name)
	tmpl, err := s.scanTemplate(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStorageTemplateNotFoundError(name)
		}
		return nil, &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    name,
			Cause:   err,
		}
	}

	return tmpl, nil
}

// GetByID retrieves a specific template version by ID.
func (s *PostgresStorage) GetByID(ctx context.Context, id TemplateID) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	query := fmt.Sprintf(`
		SELECT id, name, source, version, metadata, inference_config,
		       created_at, updated_at, created_by, tenant_id, tags
		FROM %s
		WHERE id = $1`, s.tableName())

	row := s.db.QueryRowContext(ctx, query, string(id))
	tmpl, err := s.scanTemplate(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStorageTemplateNotFoundError(string(id))
		}
		return nil, &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    string(id),
			Cause:   err,
		}
	}

	return tmpl, nil
}

// GetVersion retrieves a specific version of a template.
func (s *PostgresStorage) GetVersion(ctx context.Context, name string, version int) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	query := fmt.Sprintf(`
		SELECT id, name, source, version, metadata, inference_config,
		       created_at, updated_at, created_by, tenant_id, tags
		FROM %s
		WHERE name = $1 AND version = $2`, s.tableName())

	row := s.db.QueryRowContext(ctx, query, name, version)
	tmpl, err := s.scanTemplate(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStorageVersionNotFoundError(name, version)
		}
		return nil, &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    name,
			Version: version,
			Cause:   err,
		}
	}

	return tmpl, nil
}

// Save stores a template, creating a new version if one exists.
func (s *PostgresStorage) Save(ctx context.Context, tmpl *StoredTemplate) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if tmpl.Name == "" {
		return &StorageError{Message: ErrMsgInvalidTemplateName}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	// Begin transaction with SERIALIZABLE isolation for version safety
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresTransactionFailed,
			Name:    tmpl.Name,
			Cause:   err,
		}
	}
	defer func() { _ = tx.Rollback() }()

	// Get current max version
	var maxVersion sql.NullInt64
	err = tx.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COALESCE(MAX(version), 0) FROM %s WHERE name = $1", s.tableName()),
		tmpl.Name).Scan(&maxVersion)
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    tmpl.Name,
			Cause:   err,
		}
	}

	nextVersion := 1
	if maxVersion.Valid {
		nextVersion = int(maxVersion.Int64) + 1
	}

	// Generate ID and timestamps
	now := time.Now()
	newID := generateTemplateID()

	// Serialize JSONB fields
	metadataJSON, err := json.Marshal(tmpl.Metadata)
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresMarshalFailed,
			Name:    tmpl.Name,
			Cause:   err,
		}
	}

	var inferenceConfigJSON []byte
	if tmpl.InferenceConfig != nil {
		inferenceConfigJSON, err = json.Marshal(tmpl.InferenceConfig)
		if err != nil {
			return &StorageError{
				Message: ErrMsgPostgresMarshalFailed,
				Name:    tmpl.Name,
				Cause:   err,
			}
		}
	}

	tagsJSON, err := json.Marshal(tmpl.Tags)
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresMarshalFailed,
			Name:    tmpl.Name,
			Cause:   err,
		}
	}

	// Insert template
	insertQuery := fmt.Sprintf(`
		INSERT INTO %s
		(id, name, source, version, metadata, inference_config,
		 created_at, updated_at, created_by, tenant_id, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		s.tableName())

	_, err = tx.ExecContext(ctx, insertQuery,
		string(newID), tmpl.Name, tmpl.Source, nextVersion,
		metadataJSON, inferenceConfigJSON,
		now, now, nullString(tmpl.CreatedBy), nullString(tmpl.TenantID), tagsJSON)
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    tmpl.Name,
			Cause:   err,
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return &StorageError{
			Message: ErrMsgPostgresTransactionFailed,
			Name:    tmpl.Name,
			Cause:   err,
		}
	}

	// Update input template with generated values
	tmpl.ID = newID
	tmpl.Version = nextVersion
	tmpl.CreatedAt = now
	tmpl.UpdatedAt = now

	return nil
}

// Delete removes all versions of a template by name.
func (s *PostgresStorage) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	query := fmt.Sprintf("DELETE FROM %s WHERE name = $1", s.tableName())
	result, err := s.db.ExecContext(ctx, query, name)
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    name,
			Cause:   err,
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    name,
			Cause:   err,
		}
	}

	if rowsAffected == 0 {
		return NewStorageTemplateNotFoundError(name)
	}

	return nil
}

// DeleteVersion removes a specific version of a template.
func (s *PostgresStorage) DeleteVersion(ctx context.Context, name string, version int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	query := fmt.Sprintf("DELETE FROM %s WHERE name = $1 AND version = $2", s.tableName())
	result, err := s.db.ExecContext(ctx, query, name, version)
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    name,
			Version: version,
			Cause:   err,
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    name,
			Version: version,
			Cause:   err,
		}
	}

	if rowsAffected == 0 {
		return NewStorageVersionNotFoundError(name, version)
	}

	return nil
}

// List returns templates matching the query.
func (s *PostgresStorage) List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	if query == nil {
		query = &TemplateQuery{}
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	// Build dynamic query
	var conditions []string
	var args []interface{}
	argIdx := 1

	if query.TenantID != "" {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, query.TenantID)
		argIdx++
	}

	if query.CreatedBy != "" {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argIdx))
		args = append(args, query.CreatedBy)
		argIdx++
	}

	if query.NamePrefix != "" {
		conditions = append(conditions, fmt.Sprintf("name LIKE $%d", argIdx))
		args = append(args, query.NamePrefix+"%")
		argIdx++
	}

	if query.NameContains != "" {
		conditions = append(conditions, fmt.Sprintf("name LIKE $%d", argIdx))
		args = append(args, "%"+query.NameContains+"%")
		argIdx++
	}

	// Tags filter - ALL tags must match
	for _, tag := range query.Tags {
		conditions = append(conditions, fmt.Sprintf("tags @> $%d::jsonb", argIdx))
		tagJSON, _ := json.Marshal([]string{tag})
		args = append(args, string(tagJSON))
		argIdx++
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build full query
	var sqlQuery string
	if query.IncludeAllVersions {
		sqlQuery = fmt.Sprintf(`
			SELECT id, name, source, version, metadata, inference_config,
			       created_at, updated_at, created_by, tenant_id, tags
			FROM %s
			%s
			ORDER BY name ASC, version DESC`,
			s.tableName(), whereClause)
	} else {
		// Only latest version per name using DISTINCT ON
		sqlQuery = fmt.Sprintf(`
			SELECT DISTINCT ON (name) id, name, source, version, metadata, inference_config,
			       created_at, updated_at, created_by, tenant_id, tags
			FROM %s
			%s
			ORDER BY name ASC, version DESC`,
			s.tableName(), whereClause)
	}

	// Add LIMIT and OFFSET
	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
	}
	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Cause:   err,
		}
	}
	defer rows.Close()

	var results []*StoredTemplate
	for rows.Next() {
		tmpl, err := s.scanTemplateRow(rows)
		if err != nil {
			return nil, &StorageError{
				Message: ErrMsgPostgresScanFailed,
				Cause:   err,
			}
		}
		results = append(results, tmpl)
	}

	if err := rows.Err(); err != nil {
		return nil, &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Cause:   err,
		}
	}

	return results, nil
}

// Exists checks if a template with the given name exists.
func (s *PostgresStorage) Exists(ctx context.Context, name string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return false, NewStorageClosedError()
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE name = $1)", s.tableName())
	var exists bool
	err := s.db.QueryRowContext(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    name,
			Cause:   err,
		}
	}

	return exists, nil
}

// ListVersions returns all version numbers for a template.
func (s *PostgresStorage) ListVersions(ctx context.Context, name string) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	query := fmt.Sprintf("SELECT version FROM %s WHERE name = $1 ORDER BY version DESC", s.tableName())
	rows, err := s.db.QueryContext(ctx, query, name)
	if err != nil {
		return nil, &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    name,
			Cause:   err,
		}
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, &StorageError{
				Message: ErrMsgPostgresScanFailed,
				Name:    name,
				Cause:   err,
			}
		}
		versions = append(versions, v)
	}

	if err := rows.Err(); err != nil {
		return nil, &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Name:    name,
			Cause:   err,
		}
	}

	if versions == nil {
		versions = []int{}
	}

	return versions, nil
}

// Close releases database connections.
func (s *PostgresStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return &StorageError{Message: ErrMsgPostgresAlreadyClosed}
	}

	s.closed = true
	return s.db.Close()
}

// RunMigrations applies pending database migrations.
func (s *PostgresStorage) RunMigrations(ctx context.Context) error {
	// Create migrations table if not exists
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version     INTEGER PRIMARY KEY,
			applied_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			description VARCHAR(255)
		)`, s.migrationsTableName()))
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresMigrationFailed,
			Cause:   err,
		}
	}

	// Get applied migrations
	applied := make(map[int]bool)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT version FROM %s", s.migrationsTableName()))
	if err != nil {
		return &StorageError{
			Message: ErrMsgPostgresMigrationFailed,
			Cause:   err,
		}
	}
	defer rows.Close()

	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return &StorageError{
				Message: ErrMsgPostgresMigrationFailed,
				Cause:   err,
			}
		}
		applied[v] = true
	}

	// Apply migrations
	migrations := s.getMigrations()
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return &StorageError{
				Message: ErrMsgPostgresMigrationFailed,
				Cause:   err,
			}
		}

		if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
			_ = tx.Rollback()
			return &StorageError{
				Message: ErrMsgPostgresMigrationFailed,
				Cause:   fmt.Errorf("migration %d failed: %w", m.Version, err),
			}
		}

		if _, err := tx.ExecContext(ctx,
			fmt.Sprintf("INSERT INTO %s (version, description) VALUES ($1, $2)", s.migrationsTableName()),
			m.Version, m.Description); err != nil {
			_ = tx.Rollback()
			return &StorageError{
				Message: ErrMsgPostgresMigrationFailed,
				Cause:   err,
			}
		}

		if err := tx.Commit(); err != nil {
			return &StorageError{
				Message: ErrMsgPostgresMigrationFailed,
				Cause:   err,
			}
		}
	}

	return nil
}

// CurrentSchemaVersion returns the current schema version.
func (s *PostgresStorage) CurrentSchemaVersion(ctx context.Context) (int, error) {
	var version sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT MAX(version) FROM %s", s.migrationsTableName())).Scan(&version)
	if err != nil {
		return 0, &StorageError{
			Message: ErrMsgPostgresQueryFailed,
			Cause:   err,
		}
	}

	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

// postgresMigration represents a database migration.
type postgresMigration struct {
	Version     int
	Description string
	SQL         string
}

// getMigrations returns all available migrations.
func (s *PostgresStorage) getMigrations() []postgresMigration {
	return []postgresMigration{
		{
			Version:     1,
			Description: "Initial schema with templates table",
			SQL: fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					id               VARCHAR(255) PRIMARY KEY,
					name             VARCHAR(255) NOT NULL,
					source           TEXT NOT NULL,
					version          INTEGER NOT NULL DEFAULT 1,
					metadata         JSONB DEFAULT '{}',
					inference_config JSONB DEFAULT NULL,
					created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
					updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
					created_by       VARCHAR(255),
					tenant_id        VARCHAR(255),
					tags             JSONB DEFAULT '[]',
					CONSTRAINT %s_name_version_unique UNIQUE (name, version)
				);

				CREATE INDEX IF NOT EXISTS idx_%s_name ON %s(name);
				CREATE INDEX IF NOT EXISTS idx_%s_tenant_id ON %s(tenant_id) WHERE tenant_id IS NOT NULL;
				CREATE INDEX IF NOT EXISTS idx_%s_created_by ON %s(created_by) WHERE created_by IS NOT NULL;
				CREATE INDEX IF NOT EXISTS idx_%s_name_version ON %s(name, version DESC);
				CREATE INDEX IF NOT EXISTS idx_%s_tags ON %s USING GIN(tags);
				CREATE INDEX IF NOT EXISTS idx_%s_created_at ON %s(created_at DESC);

				CREATE OR REPLACE FUNCTION %s_update_updated_at_column()
				RETURNS TRIGGER AS $$
				BEGIN
					NEW.updated_at = NOW();
					RETURN NEW;
				END;
				$$ language 'plpgsql';

				DROP TRIGGER IF EXISTS %s_update_updated_at ON %s;
				CREATE TRIGGER %s_update_updated_at
					BEFORE UPDATE ON %s
					FOR EACH ROW
					EXECUTE FUNCTION %s_update_updated_at_column();
			`,
				s.tableName(),
				s.config.TablePrefix+"templates",
				s.config.TablePrefix+"templates", s.tableName(),
				s.config.TablePrefix+"templates", s.tableName(),
				s.config.TablePrefix+"templates", s.tableName(),
				s.config.TablePrefix+"templates", s.tableName(),
				s.config.TablePrefix+"templates", s.tableName(),
				s.config.TablePrefix+"templates", s.tableName(),
				s.config.TablePrefix+"templates",
				s.config.TablePrefix+"templates", s.tableName(),
				s.config.TablePrefix+"templates", s.tableName(),
				s.config.TablePrefix+"templates",
			),
		},
	}
}

// scanTemplate scans a single row into a StoredTemplate.
func (s *PostgresStorage) scanTemplate(row *sql.Row) (*StoredTemplate, error) {
	var (
		id               string
		name             string
		source           string
		version          int
		metadataJSON     []byte
		inferenceJSON    sql.NullString
		createdAt        time.Time
		updatedAt        time.Time
		createdBy        sql.NullString
		tenantID         sql.NullString
		tagsJSON         []byte
	)

	err := row.Scan(&id, &name, &source, &version, &metadataJSON, &inferenceJSON,
		&createdAt, &updatedAt, &createdBy, &tenantID, &tagsJSON)
	if err != nil {
		return nil, err
	}

	return s.unmarshalTemplate(id, name, source, version, metadataJSON, inferenceJSON,
		createdAt, updatedAt, createdBy, tenantID, tagsJSON)
}

// scanTemplateRow scans a rows result into a StoredTemplate.
func (s *PostgresStorage) scanTemplateRow(rows *sql.Rows) (*StoredTemplate, error) {
	var (
		id               string
		name             string
		source           string
		version          int
		metadataJSON     []byte
		inferenceJSON    sql.NullString
		createdAt        time.Time
		updatedAt        time.Time
		createdBy        sql.NullString
		tenantID         sql.NullString
		tagsJSON         []byte
	)

	err := rows.Scan(&id, &name, &source, &version, &metadataJSON, &inferenceJSON,
		&createdAt, &updatedAt, &createdBy, &tenantID, &tagsJSON)
	if err != nil {
		return nil, err
	}

	return s.unmarshalTemplate(id, name, source, version, metadataJSON, inferenceJSON,
		createdAt, updatedAt, createdBy, tenantID, tagsJSON)
}

// unmarshalTemplate converts scanned values into a StoredTemplate.
func (s *PostgresStorage) unmarshalTemplate(id, name, source string, version int,
	metadataJSON []byte, inferenceJSON sql.NullString,
	createdAt, updatedAt time.Time, createdBy, tenantID sql.NullString,
	tagsJSON []byte) (*StoredTemplate, error) {

	tmpl := &StoredTemplate{
		ID:        TemplateID(id),
		Name:      name,
		Source:    source,
		Version:   version,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
		if err := json.Unmarshal(metadataJSON, &tmpl.Metadata); err != nil {
			return nil, fmt.Errorf("%s: metadata: %w", ErrMsgPostgresUnmarshalFailed, err)
		}
	}

	// Unmarshal inference config
	if inferenceJSON.Valid && inferenceJSON.String != "" && inferenceJSON.String != "null" {
		var cfg InferenceConfig
		if err := json.Unmarshal([]byte(inferenceJSON.String), &cfg); err != nil {
			return nil, fmt.Errorf("%s: inference_config: %w", ErrMsgPostgresUnmarshalFailed, err)
		}
		tmpl.InferenceConfig = &cfg
	}

	// Unmarshal tags
	if len(tagsJSON) > 0 && string(tagsJSON) != "null" {
		if err := json.Unmarshal(tagsJSON, &tmpl.Tags); err != nil {
			return nil, fmt.Errorf("%s: tags: %w", ErrMsgPostgresUnmarshalFailed, err)
		}
	}

	// Handle nullable strings
	if createdBy.Valid {
		tmpl.CreatedBy = createdBy.String
	}
	if tenantID.Valid {
		tmpl.TenantID = tenantID.String
	}

	return tmpl, nil
}

// nullString converts an empty string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
