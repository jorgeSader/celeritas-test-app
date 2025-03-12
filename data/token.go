package data

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/upper/db/v4"
)

// TokenLength defines the length of generated tokens, configurable via the TOKEN_LENGTH environment variable.
var TokenLength = 26

func init() {
	if tl := os.Getenv("TOKEN_LENGTH"); tl != "" {
		if i, err := strconv.Atoi(tl); err == nil {
			TokenLength = i
		}
	}
}

// Token represents a token entity in the database.
type Token struct {
	ID        int       `db:"id,omitempty"`
	UserID    int       `db:"user_id"`
	FirstName string    `db:"first_name"`
	Email     string    `db:"email"`
	Hash      []byte    `db:"token_hash"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Expires   time.Time `db:"expiry"`
}

// Table returns the database table name for the Token model.
func (t *Token) Table() string {
	return "tokens"
}

// GetUserForToken retrieves the user associated with a given token hash.
// The token is hashed to match the stored token_hash in the database.
func (t *Token) GetUserForToken(plainText string) (User, error) {
	var token Token
	var user User
	hash := sha256.Sum256([]byte(plainText))
	row, err := upper.SQL().QueryRow("SELECT id, user_id, first_name, email, token_hash, created_at, updated_at, expiry FROM tokens WHERE token_hash = $1 LIMIT 1", hash[:])
	err = row.Scan(&token.ID, &token.UserID, &token.FirstName, &token.Email, &token.Hash, &token.CreatedAt, &token.UpdatedAt, &token.Expires)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, fmt.Errorf("no matching user found")
		}
		return user, err
	}
	collection := upper.Collection("users")
	res := collection.Find(db.Cond{"id": token.UserID})
	err = res.One(&user)
	if err != nil {
		return user, fmt.Errorf("no matching user found")
	}
	return user, nil
}

// GetTokensForUser retrieves all tokens associated with a given user ID.
func (t *Token) GetTokensForUser(id int) ([]*Token, error) {
	var tokens []*Token
	collection := upper.Collection(t.Table())
	res := collection.Find(db.Cond{"user_id": id})
	err := res.All(&tokens)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// Get retrieves a token by its ID.
func (t *Token) Get(id int) (*Token, error) {
	var token Token
	collection := upper.Collection(t.Table())
	res := collection.Find(db.Cond{"id =": id})
	err := res.One(&token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// GetByToken retrieves a token by its plaintext value.
// It hashes the token to match against the stored hash.
func (t *Token) GetByToken(plainText string) (*Token, error) {
	var token Token
	hash := sha256.Sum256([]byte(plainText))
	row, err := upper.SQL().QueryRow("SELECT id, user_id, first_name, email, token_hash, created_at, updated_at, expiry FROM tokens WHERE token_hash = $1 LIMIT 1", hash[:])
	if err != nil {
		return nil, err
	}
	err = row.Scan(&token.ID, &token.UserID, &token.FirstName, &token.Email, &token.Hash, &token.CreatedAt, &token.UpdatedAt, &token.Expires)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// Delete removes a token from the database by its ID.
func (t *Token) Delete(id int) error {
	collection := upper.Collection(t.Table())
	res := collection.Find(db.Cond{"id =": id})
	err := res.Delete()
	if err != nil {
		return err
	}
	return nil
}

// DeleteByToken removes a token from the database based on its plaintext value.
func (t *Token) DeleteByToken(plainText string) error {
	hash := sha256.Sum256([]byte(plainText))
	_, err := upper.SQL().Exec("DELETE FROM tokens WHERE token_hash = $1", hash[:])
	return err
}

// Insert adds a new token to the database for a user.
// It deletes existing tokens for the user first, then inserts the new one using the provided plaintext.
func (t *Token) Insert(token Token, user User, plainText string) error {
	collection := upper.Collection(t.Table())
	res := collection.Find(db.Cond{"user_id =": user.ID})
	err := res.Delete()
	if err != nil {
		return err
	}

	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()
	token.UserID = user.ID
	token.FirstName = user.FirstName
	token.Email = user.Email
	hash := sha256.Sum256([]byte(plainText))
	token.Hash = hash[:]

	_, err = collection.Insert(token)
	if err != nil {
		return err
	}
	return nil
}

// GenerateToken creates a new token for a user with a specified time-to-live (TTL).
// It returns the token struct and its plaintext value.
func (t *Token) GenerateToken(userID int, ttl time.Duration) (*Token, string, error) {
	token := &Token{
		UserID:  userID,
		Expires: time.Now().Add(ttl),
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, "", err
	}

	plainText := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	if len(plainText) < TokenLength {
		plainText = plainText + strings.Repeat("A", TokenLength-len(plainText))
	} else if len(plainText) > TokenLength {
		plainText = plainText[:TokenLength]
	}

	hash := sha256.Sum256([]byte(plainText))
	token.Hash = hash[:]
	return token, plainText, nil
}

// AuthenticateToken validates a token from an HTTP request’s Authorization header.
// It returns the associated user if the token is valid and not expired.
func (t *Token) AuthenticateToken(r *http.Request) (*User, error) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("no authorization header received")
	}

	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("invalid authorization header format")
	}

	token := headerParts[1]
	if len(token) != TokenLength {
		return nil, errors.New("invalid token length")
	}

	tok, err := t.GetByToken(token)
	if err != nil {
		return nil, errors.New("no matching token found")
	}

	if tok.Expires.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}

	user, err := t.GetUserForToken(token)
	if err != nil {
		return nil, errors.New("no matching user found for token")
	}

	return &user, nil
}

// ValidToken checks if a token is valid and not expired.
// It returns true if valid, false otherwise, with an error on failure.
func (t *Token) ValidToken(plainText string) (bool, error) {
	_, err := t.GetUserForToken(plainText)
	if err != nil {
		return false, err
	}

	token, err := t.GetByToken(plainText)
	if err != nil {
		return false, err
	}

	if token.Expires.Before(time.Now()) {
		return false, errors.New("token has expired")
	}

	return true, nil
}