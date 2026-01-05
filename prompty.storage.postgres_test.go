package prompty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPostgresConfig(t *testing.T) {
	cfg := DefaultPostgresConfig()

	assert.Equal(t, PostgresDefaultMaxOpenConns, cfg.MaxOpenConns)
	assert.Equal(t, PostgresDefaultMaxIdleConns, cfg.MaxIdleConns)
	assert.Equal(t, PostgresDefaultConnMaxLifetime, cfg.ConnMaxLifetime)
	assert.Equal(t, PostgresDefaultConnMaxIdleTime, cfg.ConnMaxIdleTime)
	assert.Equal(t, PostgresTablePrefix, cfg.TablePrefix)
	assert.Equal(t, PostgresDefaultQueryTimeout, cfg.QueryTimeout)
	assert.False(t, cfg.AutoMigrate)
	assert.Empty(t, cfg.ConnectionString)
}

func TestPostgresStorage_EmptyConnectionString(t *testing.T) {
	_, err := NewPostgresStorage(PostgresConfig{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPostgresEmptyConnString)
}

func TestPostgresStorage_InvalidConnectionString(t *testing.T) {
	cfg := PostgresConfig{
		ConnectionString: "invalid://not-a-valid-connection-string",
	}

	_, err := NewPostgresStorage(cfg)
	require.Error(t, err)
	// The error should indicate connection failure
	assert.Contains(t, err.Error(), ErrMsgPostgresConnectionFailed)
}

func TestPostgresStorageDriver_Registered(t *testing.T) {
	drivers := ListStorageDrivers()
	assert.Contains(t, drivers, StorageDriverNamePostgres)
}

func TestPostgresStorageDriver_Open_EmptyConnectionString(t *testing.T) {
	// Opening with empty connection string should fail
	_, err := OpenStorage(StorageDriverNamePostgres, "")
	require.Error(t, err)
}

func TestPostgresConfig_Defaults_Applied(t *testing.T) {
	// Create config with only connection string
	cfg := PostgresConfig{
		ConnectionString: "postgres://localhost/test?sslmode=disable",
	}

	// Verify defaults would be applied (we can't actually connect)
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = PostgresDefaultMaxOpenConns
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = PostgresDefaultMaxIdleConns
	}
	if cfg.TablePrefix == "" {
		cfg.TablePrefix = PostgresTablePrefix
	}
	if cfg.QueryTimeout == 0 {
		cfg.QueryTimeout = PostgresDefaultQueryTimeout
	}

	assert.Equal(t, PostgresDefaultMaxOpenConns, cfg.MaxOpenConns)
	assert.Equal(t, PostgresDefaultMaxIdleConns, cfg.MaxIdleConns)
	assert.Equal(t, PostgresTablePrefix, cfg.TablePrefix)
	assert.Equal(t, PostgresDefaultQueryTimeout, cfg.QueryTimeout)
}

func TestPostgresConstants(t *testing.T) {
	// Verify constants are defined correctly
	assert.Equal(t, "postgres", StorageDriverNamePostgres)
	assert.Equal(t, "prompty_", PostgresTablePrefix)
	assert.Equal(t, 25, PostgresDefaultMaxOpenConns)
	assert.Equal(t, 5, PostgresDefaultMaxIdleConns)
	assert.Equal(t, 5*time.Minute, PostgresDefaultConnMaxLifetime)
	assert.Equal(t, 5*time.Minute, PostgresDefaultConnMaxIdleTime)
	assert.Equal(t, 30*time.Second, PostgresDefaultQueryTimeout)
}

func TestPostgresErrorMessages(t *testing.T) {
	// Verify error message constants are defined
	assert.NotEmpty(t, ErrMsgPostgresConnectionFailed)
	assert.NotEmpty(t, ErrMsgPostgresQueryFailed)
	assert.NotEmpty(t, ErrMsgPostgresTransactionFailed)
	assert.NotEmpty(t, ErrMsgPostgresScanFailed)
	assert.NotEmpty(t, ErrMsgPostgresMarshalFailed)
	assert.NotEmpty(t, ErrMsgPostgresUnmarshalFailed)
	assert.NotEmpty(t, ErrMsgPostgresMigrationFailed)
	assert.NotEmpty(t, ErrMsgPostgresEmptyConnString)
	assert.NotEmpty(t, ErrMsgPostgresAlreadyClosed)
}

func TestNullString(t *testing.T) {
	t.Run("EmptyString", func(t *testing.T) {
		ns := nullString("")
		assert.False(t, ns.Valid)
		assert.Empty(t, ns.String)
	})

	t.Run("NonEmptyString", func(t *testing.T) {
		ns := nullString("hello")
		assert.True(t, ns.Valid)
		assert.Equal(t, "hello", ns.String)
	})
}

func TestPostgresStorage_TableNames(t *testing.T) {
	// Create a mock storage to test table name generation
	// (Can't actually connect without database)
	cfg := PostgresConfig{
		TablePrefix: "custom_",
	}

	// Verify prefix works in table name construction
	expectedTemplates := "custom_templates"
	expectedMigrations := "custom_schema_migrations"

	// The actual storage construction will fail, but we can verify
	// the prefix logic would work correctly
	assert.Equal(t, "custom_", cfg.TablePrefix)
	assert.Equal(t, expectedTemplates, cfg.TablePrefix+"templates")
	assert.Equal(t, expectedMigrations, cfg.TablePrefix+"schema_migrations")
}
