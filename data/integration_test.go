//go:build integration

package data

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"net/http"
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

var models Models
var testDB *sql.DB
var container testcontainers.Container

// init sets up the container runtime environment for integration tests.
func init() {
	switch os.Getenv("CONTAINER_RUNTIME") {
	case "docker":
		os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
		os.Unsetenv("TESTCONTAINERS_RYUK_DISABLED")
	default:
		os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
		uid := os.Getuid()
		if err := os.Setenv("DOCKER_HOST", fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", uid)); err != nil {
			log.Printf("Warning: Could not set DOCKER_HOST: %s (tests may fail)", err)
		}
	}
}

// TestMain handles the setup and teardown for integration tests, including starting and stopping the PostgreSQL container.
func TestMain(m *testing.M) {
	if os.Getenv("DOCKER_HOST") == "" {
		log.Printf("DOCKER_HOST not set, required for container runtime integration")
		os.Exit(1)
	}

	os.Setenv("DATABASE_TYPE", "postgres")

	// Enable upper/db logging by setting UPPER_DB_LOG to DEBUG
	os.Setenv("UPPER_DB_LOG", "DEBUG")

	ctx := context.Background()

	timeout := 30 * time.Second
	if t, err := time.ParseDuration(os.Getenv("CONTAINER_STARTUP_TIMEOUT")); err == nil {
		timeout = t
	}

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
		}).WithStartupTimeout(timeout),
	}

	var err error
	container, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Printf("Could not start container: %s", err)
		os.Exit(1)
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		log.Printf("Could not get mapped port: %s", err)
		os.Exit(1)
	}

	testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port.Port(), user, password, dbName))
	if err != nil {
		_ = container.Terminate(ctx)
		log.Printf("Could not open database connection: %s", err)
		os.Exit(1)
	}

	if err = setupTables(testDB); err != nil {
		_ = container.Terminate(ctx)
		log.Printf("Could not setup tables: %s", err)
		os.Exit(1)
	}

	models = New(testDB)

	code := m.Run()

	var cleanupErrs []error
	if err := container.Terminate(ctx); err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("could not terminate container: %v", err))
	}
	if err := testDB.Close(); err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("could not close database: %v", err))
	}
	if len(cleanupErrs) > 0 {
		log.Printf("Cleanup errors: %v", cleanupErrs)
		os.Exit(1)
	}

	os.Exit(code)
}

// setupTables creates the necessary database tables and triggers for testing.
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

		DROP TABLE IF EXISTS tokens;
		CREATE TABLE tokens (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			first_name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			token VARCHAR(255),  -- Only for legacy; not used now
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

// TestUser_Table checks if the User model returns the correct table name.
func TestUser_Table(t *testing.T) {
	s := models.Users.Table()
	if s != "users" {
		t.Errorf("got %q, want %q", s, "users")
	}
}

// TestUser_Insert tests inserting a new user into the database.
func TestUser_Insert(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "test@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	if id == 0 {
		t.Fatal("failed to insert user: id should not be zero")
	}
}

// TestUser_Insert_DuplicateEmail tests inserting a user with a duplicate email.
func TestUser_Insert_DuplicateEmail(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "duplicate@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert first user: %v", err)
	}
	_, err = models.Users.Insert(user)
	if err == nil {
		t.Fatal("expected error on duplicate email insert")
	}
	// Cleanup
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestUser_GetAll tests retrieving all users from the database.
func TestUser_GetAll(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "getall@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	users, err := models.Users.GetAll()
	if err != nil {
		t.Fatalf("failed to get users: %v", err)
	}
	if len(users) == 0 {
		t.Fatal("expected at least one user")
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestUser_GetByEmail tests retrieving a user by their email.
func TestUser_GetByEmail(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "getbyemail@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	u, err := models.Users.GetByEmail(user.Email)
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}
	if u.Email != user.Email {
		t.Fatalf("expected email %v, got %v", user.Email, u.Email)
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestUser_Get tests retrieving a user by their ID.
func TestUser_Get(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "get@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	u, err := models.Users.Get(id)
	if err != nil {
		t.Fatalf("failed to get user by id: %v", err)
	}
	if u.ID != id {
		t.Fatalf("expected id %d, got %d", id, u.ID)
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestUser_Update tests updating an existing user.
func TestUser_Update(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "update@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	u, err := models.Users.Get(id)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	u.LastName = "Smith"
	err = models.Users.Update(*u)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}
	u, err = models.Users.Get(id)
	if err != nil {
		t.Fatalf("failed to get updated user: %v", err)
	}
	if u.LastName != "Smith" {
		t.Fatalf("expected last name 'Smith', got %v", u.LastName)
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestUser_PasswordMatches tests the password matching functionality.
func TestUser_PasswordMatches(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "password@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	u, err := models.Users.Get(id)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	matches, err := u.PasswordMatches("Test@123")
	if err != nil {
		t.Fatalf("error checking password match: %v", err)
	}
	if !matches {
		t.Fatal("expected password to match")
	}
	matches, err = u.PasswordMatches("wrong")
	if err != nil {
		t.Fatalf("error checking password match: %v", err)
	}
	if matches {
		t.Fatal("expected password not to match")
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestUser_ResetPassword tests resetting a user's password.
func TestUser_ResetPassword(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "reset@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	err = models.Users.ResetPassword(id, "New@123")
	if err != nil {
		t.Fatalf("failed to reset password: %v", err)
	}
	u, err := models.Users.Get(id)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	matches, err := u.PasswordMatches("New@123")
	if err != nil {
		t.Fatalf("error checking new password: %v", err)
	}
	if !matches {
		t.Fatal("expected new password to match")
	}
	err = models.Users.ResetPassword(999, "New@123")
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestUser_Delete tests deleting a user from the database.
func TestUser_Delete(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "delete@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	err = models.Users.Delete(id)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}
	_, err = models.Users.Get(id)
	if err == nil {
		t.Fatal("expected error when getting deleted user")
	}
}

// TestToken_Table checks if the Token model returns the correct table name.
func TestToken_Table(t *testing.T) {
	s := models.Tokens.Table()
	if s != "tokens" {
		t.Errorf("got %q, want %q", s, "tokens")
	}
}

// TestToken_GenerateToken tests generating a new token.
func TestToken_GenerateToken(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "generate@example.com",
		Password:  "Test@123",
	}
	userID, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	token, err := models.Tokens.GenerateToken(userID, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if token.UserID != userID {
		t.Errorf("got user ID %d, want %d", token.UserID, userID)
	}
	if len(token.plainText) != TokenLength {
		t.Fatalf("expected token length %d, got %d", TokenLength, len(token.plainText))
	}
	if err := models.Users.Delete(userID); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestToken_Insert tests inserting a new token for a user.
func TestToken_Insert(t *testing.T) {
	user := User{FirstName: "Test", LastName: "User", Active: 1, Email: "inserttoken@example.com", Password: "Test@123"}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	user.ID = id

	token, err := models.Tokens.GenerateToken(id, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	expectedHash := sha256.Sum256([]byte(token.plainText))

	err = models.Tokens.Insert(*token, user)
	if err != nil {
		t.Fatalf("failed to insert token: %v", err)
	}

	var dbHash []byte
	query := "SELECT token_hash FROM tokens WHERE user_id = $1"
	err = testDB.QueryRow(query, id).Scan(&dbHash)
	if err != nil {
		t.Fatalf("failed to query stored hash: %v", err)
	}
	if !bytes.Equal(dbHash, expectedHash[:]) {
		t.Errorf("stored hash doesn’t match expected hash\nstored: %x\nexpected: %x", dbHash, expectedHash[:])
	}

	tok, err := models.Tokens.GetByToken(token.plainText)
	if err != nil {
		t.Fatalf("failed to get inserted token: %v", err)
	}
	if !bytes.Equal(tok.Hash, expectedHash[:]) {
		t.Errorf("retrieved hash doesn’t match expected hash\nretrieved: %x\nexpected: %x", tok.Hash, expectedHash[:])
	}
	if tok.UserID != id {
		t.Fatalf("expected user ID %d, got %d", id, tok.UserID)
	}

	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestToken_GetUserForToken tests retrieving a user for a given token.
func TestToken_GetUserForToken(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "getuser@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	user.ID = id // Set the ID

	token, err := models.Tokens.GenerateToken(id, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	err = models.Tokens.Insert(*token, user)
	if err != nil {
		t.Fatalf("failed to insert token: %v", err)
	}
	u, err := models.Tokens.GetUserForToken(token.plainText)
	if err != nil {
		t.Fatalf("failed to get user for token: %v", err)
	}
	if u.ID != id {
		t.Fatalf("expected user ID %d, got %d", id, u.ID)
	}
	_, err = models.Tokens.GetUserForToken("invalid")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestToken_GetTokensForUser tests retrieving all tokens for a user.
func TestToken_GetTokensForUser(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "gettokens@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	user.ID = id // Set the ID

	token, err := models.Tokens.GenerateToken(id, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	err = models.Tokens.Insert(*token, user)
	if err != nil {
		t.Fatalf("failed to insert token: %v", err)
	}
	tokens, err := models.Tokens.GetTokensForUser(id)
	if err != nil {
		t.Fatalf("failed to get tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	tokens, err = models.Tokens.GetTokensForUser(999)
	if err != nil {
		t.Fatalf("unexpected error for non-existent user: %v", err)
	}
	if len(tokens) > 0 {
		t.Fatal("expected no tokens for non-existent user")
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestToken_GetByToken tests retrieving a token by its plaintext value.
func TestToken_GetByToken(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "getbytoken@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	user.ID = id // Set the ID

	token, err := models.Tokens.GenerateToken(id, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	err = models.Tokens.Insert(*token, user)
	if err != nil {
		t.Fatalf("failed to insert token: %v", err)
	}
	tok, err := models.Tokens.GetByToken(token.plainText)
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}
	if tok.UserID != id {
		t.Fatalf("expected user ID %d, got %d", id, tok.UserID)
	}
	_, err = models.Tokens.GetByToken("invalid")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestToken_Get tests retrieving a token by its ID.
func TestToken_Get(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "gettoken@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	user.ID = id // Set the ID

	token, err := models.Tokens.GenerateToken(id, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	err = models.Tokens.Insert(*token, user)
	if err != nil {
		t.Fatalf("failed to insert token: %v", err)
	}
	tok, err := models.Tokens.GetByToken(token.plainText)
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}
	tokByID, err := models.Tokens.Get(tok.ID)
	if err != nil {
		t.Fatalf("failed to get token by ID: %v", err)
	}
	if tokByID.ID != tok.ID {
		t.Fatalf("expected token ID %d, got %d", tok.ID, tokByID.ID)
	}
	_, err = models.Tokens.Get(999)
	if err == nil {
		t.Fatal("expected error for non-existent token ID")
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestToken_DeleteByToken tests deleting a token by its plaintext value.
func TestToken_DeleteByToken(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "deletebytoken@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	user.ID = id // Set the ID

	token, err := models.Tokens.GenerateToken(id, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	err = models.Tokens.Insert(*token, user)
	if err != nil {
		t.Fatalf("failed to insert token: %v", err)
	}
	err = models.Tokens.DeleteByToken(token.plainText)
	if err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}
	_, err = models.Tokens.GetByToken(token.plainText)
	if err == nil {
		t.Fatal("expected error for deleted token")
	}
	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestToken_ValidToken tests the token validation functionality.
func TestToken_ValidToken(t *testing.T) {
	user := User{
		FirstName: "Test",
		LastName:  "User",
		Active:    1,
		Email:     "validtoken@example.com",
		Password:  "Test@123",
	}
	id, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	user.ID = id // Set the ID

	tests := []struct {
		name      string
		tokenTTL  time.Duration
		wantValid bool
		wantErr   string
	}{
		{"valid", 24 * time.Hour, true, ""},
		{"expired", -24 * time.Hour, false, "token has expired"},
		{"invalid", 24 * time.Hour, false, "no matching user found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var plainText string
			if tt.name != "invalid" {
				token, err := models.Tokens.GenerateToken(id, tt.tokenTTL)
				if err != nil {
					t.Fatalf("failed to generate token: %v", err)
				}
				err = models.Tokens.Insert(*token, user)
				if err != nil {
					t.Fatalf("failed to insert token: %v", err)
				}
				plainText = token.plainText
			} else {
				plainText = "invalidtoken12345678901234"
			}

			isValid, err := models.Tokens.ValidToken(plainText)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if isValid != tt.wantValid {
				t.Fatalf("expected valid=%v, got %v", tt.wantValid, isValid)
			}
		})
	}

	if err := models.Users.Delete(id); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestToken_AuthenticateToken tests the AuthenticateToken method of the Token model.
// It verifies token authentication from an HTTP request's Authorization header across multiple scenarios:
//   - Valid token: Successfully authenticates a non-expired token and returns the associated user.
//   - No Authorization header: Fails with "no authorization header received".
//   - Invalid header format: Fails with "invalid authorization header format" for missing "Bearer" or malformed headers.
//   - Invalid token length: Fails with "invalid token length" for tokens not matching TokenLength.
//   - Non-existent token: Fails with "no matching token found" for valid-length but unmatched tokens.
//   - Expired token: Fails with "token has expired" for tokens past their expiry.
//
// The test creates separate users for valid and expired tokens to avoid overwriting, inserts tokens into the database,
// and validates the expected user or error response for each case. Cleanup removes all test data from the database.
func TestToken_AuthenticateToken(t *testing.T) {
	// Insert a test user
	user := User{
		FirstName: "Auth",
		LastName:  "Test",
		Active:    1,
		Email:     "auth@example.com",
		Password:  "Auth@123",
	}
	userID, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	user.ID = userID
	t.Logf("Inserted user ID: %d", userID)

	// Generate and insert a valid token
	validToken, err := models.Tokens.GenerateToken(userID, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate valid token: %v", err)
	}
	validPlainText := validToken.plainText

	t.Logf("Valid plaintext token: %s (length: %d)", validPlainText, len(validPlainText))
	err = models.Tokens.Insert(*validToken, user)
	if err != nil {
		t.Fatalf("failed to insert valid token: %v", err)
	}
	t.Log("Valid token inserted")

	// Verify insertion
	tok, err := models.Tokens.GetByToken(validPlainText)
	if err != nil {
		t.Fatalf("failed to verify token insertion: %v", err)
	}
	t.Logf("Verified token ID: %d, Hash: %x", tok.ID, tok.Hash)

	// Check all tokens for user
	tokens, err := models.Tokens.GetTokensForUser(userID)
	if err != nil {
		t.Fatalf("failed to get tokens for user: %v", err)
	}
	t.Logf("Tokens for user %d after valid insert: %d", userID, len(tokens))

	// Generate and insert an expired token (separate user to avoid overwrite)
	expiredUser := User{
		FirstName: "Expired",
		LastName:  "Test",
		Active:    1,
		Email:     "expired@example.com",
		Password:  "Expired@123",
	}
	expiredUserID, err := models.Users.Insert(expiredUser)
	if err != nil {
		t.Fatalf("failed to insert expired user: %v", err)
	}
	expiredUser.ID = expiredUserID
	t.Logf("Inserted expired user ID: %d", expiredUserID)

	expiredToken, err := models.Tokens.GenerateToken(expiredUserID, -24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}
	expiredPlainText := expiredToken.plainText

	err = models.Tokens.Insert(*expiredToken, expiredUser)
	if err != nil {
		t.Fatalf("failed to insert expired token: %v", err)
	}
	t.Log("Expired token inserted")

	// Verify valid token still exists
	tok, err = models.Tokens.GetByToken(validPlainText)
	if err != nil {
		t.Fatalf("valid token no longer found after expired insert: %v", err)
	}
	t.Logf("Re-verified valid token ID: %d, Hash: %x", tok.ID, tok.Hash)

	tests := []struct {
		name       string
		authHeader string
		wantErr    string
		wantUserID int
	}{
		{
			name:       "Valid token",
			authHeader: "Bearer " + validPlainText,
			wantErr:    "",
			wantUserID: int(userID),
		},
		{
			name:       "No Authorization header",
			authHeader: "",
			wantErr:    "no authorization header received",
			wantUserID: 0,
		},
		{
			name:       "Invalid header format - no Bearer",
			authHeader: validPlainText,
			wantErr:    "invalid authorization header format",
			wantUserID: 0,
		},
		{
			name:       "Invalid header format - malformed",
			authHeader: "Bearer",
			wantErr:    "invalid authorization header format",
			wantUserID: 0,
		},
		{
			name:       "Invalid token length",
			authHeader: "Bearer short",
			wantErr:    "invalid token length",
			wantUserID: 0,
		},
		{
			name:       "Non-existent token",
			authHeader: "Bearer abcdefghijklmnopqrstuvwxyz",
			wantErr:    "no matching token found",
			wantUserID: 0,
		},
		{
			name:       "Expired token",
			authHeader: "Bearer " + expiredPlainText,
			wantErr:    "token has expired",
			wantUserID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			t.Logf("Testing with Authorization: %s", tt.authHeader)

			user, err := models.Tokens.AuthenticateToken(req)
			t.Logf("Result: user=%v, err=%v", user, err)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				if user != nil {
					t.Fatalf("expected nil user, got %+v", user)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if user == nil {
					t.Fatal("expected non-nil user, got nil")
				}
				if user.ID != tt.wantUserID {
					t.Fatalf("expected user ID %d, got %d", tt.wantUserID, user.ID)
				}
			}
		})
	}

	// Cleanup
	if err := models.Users.Delete(userID); err != nil {
		t.Errorf("cleanup failed for user %d: %v", userID, err)
	}
	if err := models.Users.Delete(expiredUserID); err != nil {
		t.Errorf("cleanup failed for expired user %d: %v", expiredUserID, err)
	}
}

// TestToken_Delete tests the Delete method of the Token model.
// It verifies that:
//   - A token can be successfully deleted by its ID, leaving it un retrievable.
//   - Deleting a non-existent token ID returns no error (as per upper/db behavior).
//
// The test inserts a user and a token, deletes the token by ID, and checks its absence,
// then attempts to delete a non-existent ID. Cleanup removes the test user from the database.
func TestToken_Delete(t *testing.T) {
	// Insert a test user
	user := User{
		FirstName: "Delete",
		LastName:  "Test",
		Active:    1,
		Email:     "delete@example.com",
		Password:  "Delete@123",
	}
	userID, err := models.Users.Insert(user)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	user.ID = userID
	t.Logf("Inserted user ID: %d", userID)

	// Generate and insert a token
	token, err := models.Tokens.GenerateToken(userID, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	err = models.Tokens.Insert(*token, user)
	if err != nil {
		t.Fatalf("failed to insert token: %v", err)
	}
	t.Logf("Inserted token with plaintext: %s", token.plainText)

	// Verify insertion and get token ID
	tok, err := models.Tokens.GetByToken(token.plainText)
	if err != nil {
		t.Fatalf("failed to verify token insertion: %v", err)
	}
	tokenID := tok.ID
	t.Logf("Token ID: %d", tokenID)

	tests := []struct {
		name    string
		id      int
		wantErr bool
	}{
		{
			name:    "Delete existing token",
			id:      tokenID,
			wantErr: false,
		},
		{
			name:    "Delete non-existent token",
			id:      999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.Tokens.Delete(tt.id)
			t.Logf("Deleting token ID %d, result: %v", tt.id, err)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			// Verify deletion for existing token case
			if tt.name == "Delete existing token" {
				_, err := models.Tokens.Get(tt.id)
				if err == nil {
					t.Fatalf("expected token ID %d to be deleted, but it was still found", tt.id)
				}
				// Check if GetByToken also fails
				_, err = models.Tokens.GetByToken(token.plainText)
				if err == nil {
					t.Fatalf("expected token with plaintext %s to be deleted, but it was still found", token.plainText)
				}
				t.Logf("Confirmed token ID %d deleted", tt.id)
			}
		})
	}

	// Cleanup
	if err := models.Users.Delete(userID); err != nil {
		t.Errorf("cleanup failed for user %d: %v", userID, err)
	}
}