//go:build integration

package data

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
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
	Email:     "Test@email.com",
	Active:    1,
	Password:  "Test@123",
}

var models Models
var testDB *sql.DB
var resource *dockertest.Resource
var pool *dockertest.Pool

func TestMain(m *testing.M) {
	os.Setenv("DATABASE_TYPE", "postgres")

	podmanSocket := fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", os.Getuid())
	os.Setenv("DOCKER_HOST", podmanSocket)

	if _, err := os.Stat(podmanSocket); os.IsNotExist(err) {
		cmd := exec.Command("podman", "system", "service", "--log-level=debug", "--time=0")
		if err := cmd.Start(); err != nil {
			log.Fatalf("Could not start Podman service: %s", err)
		}
		defer cmd.Process.Kill()
		for i := 0; i < 50; i++ {
			if _, err := os.Stat(podmanSocket); err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if _, err := os.Stat(podmanSocket); os.IsNotExist(err) {
			log.Fatalf("Podman socket %s not found after starting service", podmanSocket)
		}
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to Podman: %s", err)
	}

	// Purge any existing resources to ensure a fresh start
	pool.Purge(resource)

	resource, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "latest",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
		},
		ExposedPorts: []string{"5432/tcp"},
	}, func(config *docker.HostConfig) {
		// Ensure container is fully removed on cleanup
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start Postgres container: %s", err)
	}

	port := resource.GetPort("5432/tcp")

	err = pool.Retry(func() error {
		testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port, user, password, dbName))
		if err != nil {
			return err
		}
		return testDB.Ping()
	})
	if err != nil {
		pool.Purge(resource)
		log.Fatalf("Could not connect to Postgres: %s", err)
	}

	if err := setupTables(testDB); err != nil {
		pool.Purge(resource)
		log.Fatalf("Could not create tables: %s", err)
	}

	models = New(testDB)

	code := m.Run()

	testDB.Close()
	if err := pool.Purge(resource); err != nil {
		log.Printf("Could not purge container: %s", err)
	}

	os.Exit(code)
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

func TestUsers_Table(t *testing.T) {
	if models.Users.Table() != "users" {
		t.Errorf("expected table name 'users', got %s", models.Users.Table())
	}
}

func TestUsers_InsertAndGet(t *testing.T) {
	id, err := models.Users.Insert(testUser)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	user, err := models.Users.Get(id)
	if err != nil {
		t.Fatalf("failed to get user %d: %s", id, err)
	}

	if user.Email != testUser.Email {
		t.Errorf("expected email %s, got %s", testUser.Email, user.Email)
	}
}

func TestUsers_GetByEmail(t *testing.T) {
	id, err := models.Users.Insert(testUser)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	user, err := models.Users.GetByEmail(testUser.Email)
	if err != nil {
		t.Fatalf("failed to get user by email %s: %s", testUser.Email, err)
	}

	if user.FirstName != testUser.FirstName {
		t.Errorf("expected first name %s, got %s", testUser.FirstName, user.FirstName)
	}

	// Cleanup
	if err := models.Users.Delete(id); err != nil {
		t.Fatalf("failed to delete user %d: %s", id, err)
	}
}

func TestTokens_GenerateAndValidate(t *testing.T) {
	id, err := models.Users.Insert(testUser)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}
	defer models.Users.Delete(id) // Cleanup even on failure

	token, err := models.Tokens.GenerateToken(id, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %s", err)
	}

	if err := models.Tokens.Insert(*token, testUser); err != nil {
		t.Fatalf("failed to insert token: %s", err)
	}

	valid, err := models.Tokens.ValidToken(token.PlainText)
	if err != nil {
		t.Fatalf("failed to validate token: %s", err)
	}
	if !valid {
		t.Error("expected token to be valid")
	}
}
