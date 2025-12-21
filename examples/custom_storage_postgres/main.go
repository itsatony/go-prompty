//go:build postgres
// +build postgres

// Package main demonstrates implementing a custom PostgreSQL storage backend
// for go-prompty. This example shows the complete implementation pattern
// that can be adapted to any database (MySQL, MongoDB, Redis, etc.).
//
// To run this example:
// 1. Install PostgreSQL and create a database
// 2. Run the schema: psql your_database < schema.sql
// 3. Install the driver: go get github.com/lib/pq
// 4. Run with build tag: go run -tags=postgres .
//
// See schema.sql in this directory for the required database schema.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/itsatony/go-prompty"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// =============================================================================
// PostgreSQL Storage Implementation
// =============================================================================

// PostgresStorage implements prompty.TemplateStorage using PostgreSQL.
// This implementation is thread-safe and supports all storage operations
// including versioning, multi-tenancy, and metadata.
type PostgresStorage struct {
	db     *sql.DB
	mu     sync.RWMutex // Protects closed state
	closed bool
}

// NewPostgresStorage creates a new PostgreSQL-backed template storage.
// The connectionString should be a valid PostgreSQL connection string:
// "postgres://user:password@host:port/database?sslmode=disable"
func NewPostgresStorage(connectionString string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool for production use
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresStorage{db: db}, nil
}

// Get retrieves the latest version of a template by name.
func (s *PostgresStorage) Get(ctx context.Context, name string) (*prompty.StoredTemplate, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("storage is closed")
	}
	s.mu.RUnlock()

	query := `
		SELECT id, name, source, version, metadata, created_at, updated_at,
		       created_by, tenant_id, tags
		FROM templates
		WHERE name = $1
		ORDER BY version DESC
		LIMIT 1`

	return s.scanTemplate(s.db.QueryRowContext(ctx, query, name))
}

// GetByID retrieves a specific template by ID.
func (s *PostgresStorage) GetByID(ctx context.Context, id prompty.TemplateID) (*prompty.StoredTemplate, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("storage is closed")
	}
	s.mu.RUnlock()

	query := `
		SELECT id, name, source, version, metadata, created_at, updated_at,
		       created_by, tenant_id, tags
		FROM templates
		WHERE id = $1`

	return s.scanTemplate(s.db.QueryRowContext(ctx, query, string(id)))
}

// GetVersion retrieves a specific version of a template.
func (s *PostgresStorage) GetVersion(ctx context.Context, name string, version int) (*prompty.StoredTemplate, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("storage is closed")
	}
	s.mu.RUnlock()

	query := `
		SELECT id, name, source, version, metadata, created_at, updated_at,
		       created_by, tenant_id, tags
		FROM templates
		WHERE name = $1 AND version = $2`

	return s.scanTemplate(s.db.QueryRowContext(ctx, query, name, version))
}

// Save stores a template. If a template with the same name exists,
// a new version is created automatically.
func (s *PostgresStorage) Save(ctx context.Context, tmpl *prompty.StoredTemplate) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("storage is closed")
	}
	s.mu.RUnlock()

	// Start transaction for version management
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current max version for this template name
	var maxVersion sql.NullInt64
	err = tx.QueryRowContext(ctx,
		`SELECT MAX(version) FROM templates WHERE name = $1`,
		tmpl.Name,
	).Scan(&maxVersion)
	if err != nil {
		return fmt.Errorf("failed to get max version: %w", err)
	}

	// Calculate new version
	newVersion := 1
	if maxVersion.Valid {
		newVersion = int(maxVersion.Int64) + 1
	}

	// Generate ID if not provided
	if tmpl.ID == "" {
		tmpl.ID = prompty.TemplateID(fmt.Sprintf("tmpl_%d_%s", time.Now().UnixNano(), tmpl.Name))
	}

	// Serialize metadata and tags
	metadataJSON, err := json.Marshal(tmpl.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}
	tagsJSON, err := json.Marshal(tmpl.Tags)
	if err != nil {
		return fmt.Errorf("failed to serialize tags: %w", err)
	}

	now := time.Now().UTC()

	// Insert new version
	_, err = tx.ExecContext(ctx, `
		INSERT INTO templates (id, name, source, version, metadata, created_at,
		                       updated_at, created_by, tenant_id, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		string(tmpl.ID), tmpl.Name, tmpl.Source, newVersion, metadataJSON,
		now, now, tmpl.CreatedBy, tmpl.TenantID, tagsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert template: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update template with generated values
	tmpl.Version = newVersion
	tmpl.CreatedAt = now
	tmpl.UpdatedAt = now

	return nil
}

// Delete removes all versions of a template by name.
func (s *PostgresStorage) Delete(ctx context.Context, name string) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("storage is closed")
	}
	s.mu.RUnlock()

	result, err := s.db.ExecContext(ctx, `DELETE FROM templates WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return prompty.NewStorageTemplateNotFoundError(name)
	}

	return nil
}

// DeleteVersion removes a specific version of a template.
func (s *PostgresStorage) DeleteVersion(ctx context.Context, name string, version int) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("storage is closed")
	}
	s.mu.RUnlock()

	result, err := s.db.ExecContext(ctx,
		`DELETE FROM templates WHERE name = $1 AND version = $2`,
		name, version,
	)
	if err != nil {
		return fmt.Errorf("failed to delete template version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return prompty.NewStorageTemplateNotFoundError(name)
	}

	return nil
}

// List returns templates matching the query.
func (s *PostgresStorage) List(ctx context.Context, query *prompty.TemplateQuery) ([]*prompty.StoredTemplate, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("storage is closed")
	}
	s.mu.RUnlock()

	// Build dynamic query based on filters
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT id, name, source, version, metadata, created_at, updated_at,
		       created_by, tenant_id, tags
		FROM templates`

	if query != nil {
		if query.TenantID != "" {
			conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIndex))
			args = append(args, query.TenantID)
			argIndex++
		}

		if query.NamePrefix != "" {
			conditions = append(conditions, fmt.Sprintf("name LIKE $%d", argIndex))
			args = append(args, query.NamePrefix+"%")
			argIndex++
		}

		// For tags filtering, use JSON containment
		if len(query.Tags) > 0 {
			for _, tag := range query.Tags {
				conditions = append(conditions, fmt.Sprintf("tags @> $%d::jsonb", argIndex))
				tagJSON, _ := json.Marshal([]string{tag})
				args = append(args, string(tagJSON))
				argIndex++
			}
		}

		// If not including all versions, use a subquery to get latest only
		if !query.IncludeAllVersions {
			conditions = append(conditions, `version = (
				SELECT MAX(version) FROM templates t2 WHERE t2.name = templates.name
			)`)
		}
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY name ASC, version DESC"

	// Apply pagination
	if query != nil {
		if query.Limit > 0 {
			baseQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
		}
		if query.Offset > 0 {
			baseQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []*prompty.StoredTemplate
	for rows.Next() {
		tmpl, err := s.scanTemplateRow(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, tmpl)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return templates, nil
}

// Exists checks if a template with the given name exists.
func (s *PostgresStorage) Exists(ctx context.Context, name string) (bool, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return false, fmt.Errorf("storage is closed")
	}
	s.mu.RUnlock()

	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM templates WHERE name = $1)`,
		name,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return exists, nil
}

// ListVersions returns all version numbers for a template.
func (s *PostgresStorage) ListVersions(ctx context.Context, name string) ([]int, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("storage is closed")
	}
	s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx,
		`SELECT version FROM templates WHERE name = $1 ORDER BY version DESC`,
		name,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, rows.Err()
}

// Close releases database connections.
func (s *PostgresStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	return s.db.Close()
}

// =============================================================================
// Helper Methods
// =============================================================================

// scanTemplate scans a single row into a StoredTemplate.
func (s *PostgresStorage) scanTemplate(row *sql.Row) (*prompty.StoredTemplate, error) {
	var (
		id           string
		name         string
		source       string
		version      int
		metadataJSON []byte
		createdAt    time.Time
		updatedAt    time.Time
		createdBy    sql.NullString
		tenantID     sql.NullString
		tagsJSON     []byte
	)

	err := row.Scan(&id, &name, &source, &version, &metadataJSON, &createdAt,
		&updatedAt, &createdBy, &tenantID, &tagsJSON)
	if err == sql.ErrNoRows {
		return nil, prompty.NewStorageTemplateNotFoundError(name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan template: %w", err)
	}

	return s.buildTemplate(id, name, source, version, metadataJSON, createdAt,
		updatedAt, createdBy, tenantID, tagsJSON)
}

// scanTemplateRow scans a row from a multi-row result.
func (s *PostgresStorage) scanTemplateRow(rows *sql.Rows) (*prompty.StoredTemplate, error) {
	var (
		id           string
		name         string
		source       string
		version      int
		metadataJSON []byte
		createdAt    time.Time
		updatedAt    time.Time
		createdBy    sql.NullString
		tenantID     sql.NullString
		tagsJSON     []byte
	)

	err := rows.Scan(&id, &name, &source, &version, &metadataJSON, &createdAt,
		&updatedAt, &createdBy, &tenantID, &tagsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to scan template row: %w", err)
	}

	return s.buildTemplate(id, name, source, version, metadataJSON, createdAt,
		updatedAt, createdBy, tenantID, tagsJSON)
}

// buildTemplate constructs a StoredTemplate from scanned values.
func (s *PostgresStorage) buildTemplate(
	id, name, source string,
	version int,
	metadataJSON []byte,
	createdAt, updatedAt time.Time,
	createdBy, tenantID sql.NullString,
	tagsJSON []byte,
) (*prompty.StoredTemplate, error) {
	tmpl := &prompty.StoredTemplate{
		ID:        prompty.TemplateID(id),
		Name:      name,
		Source:    source,
		Version:   version,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if createdBy.Valid {
		tmpl.CreatedBy = createdBy.String
	}
	if tenantID.Valid {
		tmpl.TenantID = tenantID.String
	}

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &tmpl.Metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	// Parse tags JSON
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &tmpl.Tags); err != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}
	}

	return tmpl, nil
}

// =============================================================================
// Storage Driver Registration (Optional)
// =============================================================================

// PostgresDriver implements prompty.StorageDriver for driver-based opening.
type PostgresDriver struct{}

// Open creates a PostgresStorage from a connection string.
func (d *PostgresDriver) Open(connectionString string) (prompty.TemplateStorage, error) {
	return NewPostgresStorage(connectionString)
}

// Register the driver with prompty (call from init or main)
func RegisterPostgresDriver() {
	prompty.RegisterStorageDriver("postgres", &PostgresDriver{})
}

// =============================================================================
// Example Usage
// =============================================================================

func main() {
	// Get connection string from environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/prompty_test?sslmode=disable"
	}

	fmt.Println("=== PostgreSQL Custom Storage Example ===\n")

	// Check if we can actually connect
	storage, err := NewPostgresStorage(connStr)
	if err != nil {
		fmt.Printf("Note: Could not connect to PostgreSQL: %v\n", err)
		fmt.Println("\nTo run this example:")
		fmt.Println("1. Start PostgreSQL")
		fmt.Println("2. Create the database: createdb prompty_test")
		fmt.Println("3. Run the schema: psql prompty_test < schema.sql")
		fmt.Println("4. Set DATABASE_URL or use default connection")
		fmt.Println("\nShowing code structure instead...\n")
		showCodeExample()
		return
	}
	defer storage.Close()

	ctx := context.Background()

	// Create storage engine with our custom PostgreSQL storage
	engine, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
		Storage: storage,
	})
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Example 1: Save a template
	fmt.Println("--- Saving templates ---")
	err = engine.Save(ctx, &prompty.StoredTemplate{
		Name:   "greeting",
		Source: `Hello {~prompty.var name="user" default="World" /~}!`,
		Tags:   []string{"public", "greeting"},
		Metadata: map[string]string{
			"author": "demo",
		},
		TenantID: "tenant_demo",
	})
	if err != nil {
		log.Fatalf("Failed to save template: %v", err)
	}
	fmt.Println("Saved 'greeting' template (version 1)")

	// Example 2: Execute the template
	fmt.Println("\n--- Executing templates ---")
	result, err := engine.Execute(ctx, "greeting", map[string]any{
		"user": "Alice",
	})
	if err != nil {
		log.Fatalf("Failed to execute: %v", err)
	}
	fmt.Printf("Result: %s\n", result)

	// Example 3: Update template (creates version 2)
	fmt.Println("\n--- Updating template (creates new version) ---")
	err = engine.Save(ctx, &prompty.StoredTemplate{
		Name:   "greeting",
		Source: `Hi {~prompty.var name="user" default="there" /~}! Welcome!`,
		Tags:   []string{"public", "greeting", "v2"},
	})
	if err != nil {
		log.Fatalf("Failed to update template: %v", err)
	}
	fmt.Println("Updated 'greeting' template (version 2)")

	// Example 4: Get version history
	fmt.Println("\n--- Version history ---")
	versions, err := storage.ListVersions(ctx, "greeting")
	if err != nil {
		log.Fatalf("Failed to list versions: %v", err)
	}
	fmt.Printf("Available versions: %v\n", versions)

	// Example 5: Execute specific version
	result, err = engine.ExecuteVersion(ctx, "greeting", 1, map[string]any{
		"user": "Bob",
	})
	if err != nil {
		log.Fatalf("Failed to execute version: %v", err)
	}
	fmt.Printf("Version 1 result: %s\n", result)

	// Example 6: Query templates
	fmt.Println("\n--- Querying templates ---")
	templates, err := storage.List(ctx, &prompty.TemplateQuery{
		Tags: []string{"public"},
	})
	if err != nil {
		log.Fatalf("Failed to list templates: %v", err)
	}
	fmt.Printf("Found %d templates with 'public' tag\n", len(templates))

	// Cleanup
	fmt.Println("\n--- Cleanup ---")
	err = storage.Delete(ctx, "greeting")
	if err != nil {
		log.Fatalf("Failed to delete: %v", err)
	}
	fmt.Println("Deleted 'greeting' template")

	fmt.Println("\nExample complete!")
}

func showCodeExample() {
	fmt.Println(`
// =============================================================================
// How to use PostgreSQL storage in your application:
// =============================================================================

// Option 1: Direct instantiation
storage, err := NewPostgresStorage("postgres://user:pass@host/db")
if err != nil {
    log.Fatal(err)
}

engine, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: storage,
})

// Option 2: With caching for production
cachedStorage := prompty.NewCachedStorage(storage, prompty.CacheConfig{
    TTL:        5 * time.Minute,
    MaxEntries: 1000,
})

engine, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: cachedStorage,
})

// Option 3: Using driver registration
RegisterPostgresDriver()
storage, err := prompty.OpenStorage("postgres", connectionString)

// =============================================================================
// Key implementation notes:
// =============================================================================

// 1. Thread Safety: The implementation uses sync.RWMutex for the closed state
//    and relies on database/sql's built-in connection pooling for concurrent access.

// 2. Versioning: Save() automatically increments version within a transaction
//    to prevent race conditions.

// 3. Error Handling: Returns prompty.NewStorageTemplateNotFoundError() for
//    not-found cases to match the expected interface behavior.

// 4. JSON Storage: Metadata and tags are stored as JSONB for flexible querying.

// 5. Connection Pooling: Configure db.SetMaxOpenConns() etc. for your workload.

// =============================================================================
// Adapting to other databases:
// =============================================================================

// MySQL:    Change $1, $2 to ?, use different JSON functions
// MongoDB:  Use bson instead of JSON, different query patterns
// Redis:    Use hash structures, different versioning approach
// DynamoDB: Use partition/sort keys for name/version
`)
}
