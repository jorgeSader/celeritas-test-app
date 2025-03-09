//go:build integration

// run tests with this command: go test . --tags integration --count=1
package data

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	host     = "localhost"
	user     = "postgres"
	password = "secret"
	dbName   = "celeritas_test"
	dsn      = "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC connect_timeout=5"
)

var testUser = User{
	FirstName: "Some",
	LastName:  "Guy",
	Email:     "Test@test.com",
	Active:    1,
	Password:  "Test@123",
}

var models Models
var testDB *sql.DB
var container testcontainers.Container

// init sets environment variables before package initialization
func init() {
	// TODO: add ability to choose podman or Docker?
	// Set DOCKER_HOST  and TESTCONTAINERS_RYUK_DISABLED explicitly for Podman
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	uid := os.Getuid()
	if err := os.Setenv("DOCKER_HOST", fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", uid)); err != nil {
		log.Printf("Warning: Could not set DOCKER_HOST: %s (tests may fail)", err)
	}
}

func TestMain(m *testing.M) {
	// Verify DOCKER_HOST is set correctly, fail here if critical
	if os.Getenv("DOCKER_HOST") == "" {
		log.Fatalf("DOCKER_HOST not set, required for Podman integration")
	}

	os.Setenv("DATABASE_TYPE", "postgres")

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image: "postgres:latest",
		Env: map[string]string{
			"POSTGRES_USER":     user,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       dbName,
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForSQL("5432/tcp", "pgx", func(host string, port nat.Port) string {
			return fmt.Sprintf(dsn, host, port.Port(), user, password, dbName)
		}).WithStartupTimeout(30 * time.Second),
	}

	var err error
	container, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		log.Fatalf("Could not get mapped port: %s", err)
	}

	testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port.Port(), user, password, dbName))
	if err != nil {
		_ = container.Terminate(ctx)
		log.Fatalf("Could not open database connection: %s", err)
	}

	if err := setupTables(testDB); err != nil {
		_ = container.Terminate(ctx)
		log.Fatalf("Could not setup tables: %s", err)
	}

	defer func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("Could not terminate container: %s", err)
		}
		if err := testDB.Close(); err != nil {
			log.Printf("Could not close database connection: %s", err)
		}
	}()

	m.Run()
}

func setupTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE OR REPLACE FUNCTION trigger_set_timestamp()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		DROP TABLE IF EXISTS users CASCADE;
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			first_name VARCHAR(255) NOT NULL,
			last_name VARCHAR(255) NOT NULL,
			user_active INTEGER NOT NULL DEFAULT 0,
			email VARCHAR(255) NOT NULL UNIQUE,
			password VARCHAR(60) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
		CREATE TRIGGER set_timestamp
			BEFORE UPDATE ON users
			FOR EACH ROW
			EXECUTE FUNCTION trigger_set_timestamp();

		DROP TABLE IF EXISTS remember_tokens;
		CREATE TABLE remember_tokens (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			remember_token VARCHAR(100) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
		CREATE TRIGGER set_timestamp
			BEFORE UPDATE ON remember_tokens
			FOR EACH ROW
			EXECUTE FUNCTION trigger_set_timestamp();

		DROP TABLE IF EXISTS tokens;
		CREATE TABLE tokens (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			first_name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			token VARCHAR(255) NOT NULL,
			token_hash BYTEA NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			expiry TIMESTAMP NOT NULL
		);
		CREATE TRIGGER set_timestamp
			BEFORE UPDATE ON tokens
			FOR EACH ROW
			EXECUTE FUNCTION trigger_set_timestamp();
	`)
	return err
}