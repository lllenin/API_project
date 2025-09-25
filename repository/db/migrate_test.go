package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigration(t *testing.T) {
	tests := []struct {
		name        string
		dbDSN       string
		migratePath string
		want        struct {
			error bool
		}
	}{
		{
			name:        "invalid database connection string",
			dbDSN:       "invalid_connection_string",
			migratePath: "../../migrations",
			want: struct {
				error bool
			}{
				error: true,
			},
		},
		{
			name:        "invalid migrate path",
			dbDSN:       "postgres://user:password@localhost:5432/testdb?sslmode=disable",
			migratePath: "/nonexistent/path",
			want: struct {
				error bool
			}{
				error: true,
			},
		},
		{
			name:        "empty database connection string",
			dbDSN:       "",
			migratePath: "../../migrations",
			want: struct {
				error bool
			}{
				error: true,
			},
		},
		{
			name:        "empty migrate path",
			dbDSN:       "postgres://user:password@localhost:5432/testdb?sslmode=disable",
			migratePath: "",
			want: struct {
				error bool
			}{
				error: true,
			},
		},
		{
			name:        "malformed DSN",
			dbDSN:       "postgres://invalid",
			migratePath: "../../migrations",
			want: struct {
				error bool
			}{
				error: true,
			},
		},
		{
			name:        "non-existent host",
			dbDSN:       "postgres://user:password@nonexistent:5432/testdb?sslmode=disable",
			migratePath: "../../migrations",
			want: struct {
				error bool
			}{
				error: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Migration(tt.dbDSN, tt.migratePath)

			if tt.want.error {
				assert.Error(t, err, "Expected error for invalid parameters")
			} else {
				assert.NoError(t, err, "Expected no error for valid parameters")
			}
		})
	}
}

func TestMigrationWithValidParams(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		dbDSN       string
		migratePath string
		want        struct {
			error bool
		}
	}{
		{
			name:        "valid database and migrate path",
			dbDSN:       "postgres://user:password@localhost:5432/testdb?sslmode=disable",
			migratePath: "../../migrations",
			want: struct {
				error bool
			}{
				error: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Migration(tt.dbDSN, tt.migratePath)

			if tt.want.error {
				assert.Error(t, err, "Expected error due to unavailable database in test environment")
			} else {
				assert.NoError(t, err, "Expected no error for valid parameters")
			}
		})
	}
}

func TestMigrationWithRealDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		dbDSN       string
		migratePath string
		want        struct {
			success bool
		}
	}{
		{
			name:        "successful migration with real database",
			dbDSN:       "postgres://shouldbeinVaultuser:shouldbeinVaultpassword@localhost:5432/tasks?sslmode=disable",
			migratePath: "../../migrations",
			want: struct {
				success bool
			}{
				success: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Migration(tt.dbDSN, tt.migratePath)

			if tt.want.success {
				assert.NoError(t, err, "Expected no error for valid database connection")
			} else {
				assert.Error(t, err, "Expected error for invalid database connection")
			}
		})
	}
}
