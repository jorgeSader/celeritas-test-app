//go:build trevor

// run tests with this command: go test . --tags integration --count=1
package data

import (
	"database/sql"
)

const (
	host     = "localhost"
	user     = "postgres"
	password = "secret"
	dbName   = "devify_test"
	port     = "5435"
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
var resource *dockertest.Resource
var pool *dockertest.Pool

func TestMain(m *testing.M) {
	os.Setenv("DATABASE_TYPE", "postgres")

	p, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to Podman: %s", err)
	}

	pool = p

	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "latest",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
			"POSTGRES_HOST=" + host,
			"POSTGRES_PORT=" + port,
		},
		exposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}

	resource, err = pool.RunWithOptions(&opts)
	if err != nil {
		pool.Purge(resource)
		log.Fatalf("Could not start resource: %s", err)
	}

	if err := pool.Retry(func() error {
		var err error
		testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port, user, password, dbName))
		if err != nil {
			return err
		}
		return testDB.Ping()
	}); err != nil {
		pool.Purge(resource)
		log.Fatalf("Could not connect to DB: %s", err)
	}
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
