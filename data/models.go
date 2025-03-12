package data

import (
	"database/sql"
	"fmt"
	"github.com/upper/db/v4/adapter/mysql"
	"github.com/upper/db/v4/adapter/postgresql"
	"github.com/upper/db/v4/adapter/sqlite"
	"os"
	"strings"

	"github.com/upper/db/v4"
)

// database is the global SQL database connection pool.
var database *sql.DB

// upper is the global upper.io database session.
var upper db.Session

// Models encapsulates the User and Token models for database operations.
type Models struct {
	Users  User
	Tokens Token
}

// New initializes the models with the provided database pool.
// It panics if initialization fails, suitable for development. For production, consider graceful error handling.
func New(databasePool *sql.DB) Models {
	m, err := NewWithError(databasePool)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize models: %v", err))
	}
	return m
}

// NewWithError initializes the models with the provided database pool and returns an error if it fails.
// It configures the upper.io session based on the DATABASE_TYPE environment variable.
func NewWithError(databasePool *sql.DB) (Models, error) {
	if databasePool == nil {
		return Models{}, fmt.Errorf("database pool is nil")
	}
	database = databasePool
	dbType := strings.ToLower(os.Getenv("DATABASE_TYPE"))
	if dbType == "" {
		return Models{}, fmt.Errorf("DATABASE_TYPE environment variable not set")
	}

	switch dbType {
	case "mysql", "mariadb":
		session, err := mysql.New(databasePool)
		if err != nil {
			return Models{}, err
		}
		upper = session
	case "postgresql", "postgres":
		session, err := postgresql.New(databasePool)
		if err != nil {
			return Models{}, err
		}
		upper = session
	case "sqlite", "turso", "libsql":
		session, err := sqlite.New(databasePool)
		if err != nil {
			return Models{}, err
		}
		upper = session
	case "mongo", "mongodb":
		return Models{}, fmt.Errorf("mongo not implemented")
	default:
		return Models{}, fmt.Errorf("unknown DATABASE_TYPE: %s", dbType)
	}

	return Models{
		Users:  User{},
		Tokens: Token{},
	}, nil
}

// GetInsertID converts a db.ID to an integer for use as a record identifier.
// It supports int and int64 types, returning the value as an int. For unsupported
// types like strings (e.g., UUIDs), it returns 0, assuming the caller handles such cases.
func GetInsertID(i db.ID) int {
	switch v := i.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case string:
		return 0 // Suitable for UUIDs or non-numeric IDs in this context
	default:
		return 0 // Default case for unrecognized types
	}
}