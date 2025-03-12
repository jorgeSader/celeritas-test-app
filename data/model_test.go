package data

import (
	"database/sql"
	"fmt"
	"github.com/upper/db/v4"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestNew tests the NewWithError function for initializing Models with various database types.
func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		dbType      string
		dbPool      *sql.DB
		mockSetup   func(sqlmock.Sqlmock)
		wantErr     bool
		errContains string
	}{
		{
			name:   "PostgresSuccess",
			dbType: "postgres",
			dbPool: nil, // Will be set in test
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT CURRENT_DATABASE\(\) AS name`).
					WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("testdb"))
			},
			wantErr: false,
		},
		{
			name:   "MySQLSuccess",
			dbType: "mysql",
			dbPool: nil, // Will be set in test
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT DATABASE\(\) AS name`).
					WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("testdb"))
			},
			wantErr: false,
		},
		{
			name:        "NilDB",
			dbType:      "postgres",
			dbPool:      nil,
			mockSetup:   func(sqlmock.Sqlmock) {}, // No mock setup needed
			wantErr:     true,
			errContains: "database pool is nil",
		},
		{
			name:        "InvalidDBType",
			dbType:      "invalid",
			dbPool:      sqlmockDB(),
			mockSetup:   func(sqlmock.Sqlmock) {},
			wantErr:     true,
			errContains: "unknown DATABASE_TYPE: invalid",
		},
		{
			name:        "EmptyDBType",
			dbType:      "",
			dbPool:      sqlmockDB(),
			mockSetup:   func(sqlmock.Sqlmock) {},
			wantErr:     true,
			errContains: "DATABASE_TYPE environment variable not set",
		},
		{
			name:        "MongoNotImplemented",
			dbType:      "mongo",
			dbPool:      sqlmockDB(),
			mockSetup:   func(sqlmock.Sqlmock) {},
			wantErr:     true,
			errContains: "mongo not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DATABASE_TYPE", tt.dbType)
			defer os.Unsetenv("DATABASE_TYPE")

			var mock sqlmock.Sqlmock
			if tt.dbPool == nil && tt.mockSetup != nil && !tt.wantErr { // Only setup mock for success cases
				db, m, err := sqlmock.New()
				if err != nil {
					t.Fatalf("failed to create sqlmock: %v", err)
				}
				defer db.Close()
				tt.dbPool = db
				mock = m
				tt.mockSetup(mock)
			}

			_, err := NewWithError(tt.dbPool) // Fixed: Using NewWithError directly
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWithError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" && (err == nil || !strings.Contains(err.Error(), tt.errContains)) {
				t.Errorf("NewWithError() error = %v, want error containing %q", err, tt.errContains)
			}
			if mock != nil {
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("unfulfilled expectations: %v", err)
				}
			}
		})
	}
}

// sqlmockDB creates a new sqlmock database connection for testing.
// It panics if the mock creation fails.
func sqlmockDB() *sql.DB {
	db, _, err := sqlmock.New()
	if err != nil {
		panic(fmt.Sprintf("failed to create sqlmock: %v", err))
	}
	return db
}

// TestGetInsertID tests the GetInsertID function with various input types.
func TestGetInsertID(t *testing.T) {
	tests := []struct {
		name    string
		id      db.ID
		wantID  int
		wantErr bool
	}{
		{"Int", 1, 1, false},
		{"Int64", int64(2), 2, false},
		{"StringUUID", "550e8400-e29b-41d4-a716-446655440000", 0, true},
		{"FloatUnsupported", 3.14, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID := GetInsertID(tt.id) // Note: Function doesn’t return error, adjusting test accordingly
			if gotID != tt.wantID {
				t.Errorf("GetInsertID() = %v, want %v", gotID, tt.wantID)
			}
			// Since GetInsertID doesn’t return an error, we can’t test wantErr directly
			// Future improvement: Modify GetInsertID to return an error for unsupported types
		})
	}
}