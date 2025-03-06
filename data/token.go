package data

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"net/http"
	"strings"
	"time"

	up "github.com/upper/db/v4"
)

// Token represents an authentication token associated with a user.
type Token struct {
	ID        int       `db:"id" json:"id"`
	UserID    int       `db:"user_id" json:"user_id"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Email     string    `db:"email" json:"email"`
	PlainText string    `db:"token" json:"plain_text"`
	Hash      []byte    `db:"token_hash" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Expires   time.Time `db:"expiry" json:"expiry"`
}

// Table returns the database table name for tokens.
func (t *Token) Table() string {
	return "tokens"
}

// GetUserForToken retrieves the user for a given plaintext token.
func (t *Token) GetUserForToken(token string) (*User, error) {
	var user User
	var theToken Token

	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"token": token})
	err := res.One(&theToken)
	if err != nil {
		return nil, err
	}

	collection = upper.Collection(user.Table())
	res = collection.Find(up.Cond{"id": theToken.UserID})
	err = res.One(&user)
	if err != nil {
		return nil, err
	}

	user.Token = theToken
	return &user, nil
}

// GetTokensForUser retrieves all tokens for a user by ID.
func (t *Token) GetTokensForUser(id int) ([]*Token, error) {
	var tokens []*Token

	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"user_id": id})
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
	res := collection.Find(up.Cond{"id =": id})
	err := res.One(&token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// GetByToken retrieves a token by its plaintext value.
func (t *Token) GetByToken(plainText string) (*Token, error) {
	var token Token

	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"token =": plainText})
	err := res.One(&token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// Delete removes a token by its ID.
func (t *Token) Delete(id int) error {
	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"id =": id})
	err := res.Delete()
	if err != nil {
		return err
	}
	return nil
}

// DeleteByToken removes a token by its plaintext value.
func (t *Token) DeleteByToken(plainText string) error {
	collection := upper.Collection(t.Table())
	res := collection.Find(up.Cond{"token =": plainText})
	err := res.Delete()
	if err != nil {
		return err
	}
	return nil
}

// Insert adds a new token for a user, replacing existing tokens.
func (t *Token) Insert(token Token, user User) error {
	collection := upper.Collection(t.Table())

	// Delete existing tokens for the user.
	res := collection.Find(up.Cond{"user_id =": user.ID})
	err := res.Delete()
	if err != nil {
		return err
	}

	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()
	token.UserID = user.ID
	token.FirstName = user.FirstName
	token.Email = user.Email

	_, err = collection.Insert(token)
	if err != nil {
		return err
	}
	return nil
}

// GenerateToken creates a new token for a user with a specified TTL.
func (t *Token) GenerateToken(userID int, ttl time.Duration) (*Token, error) {
	token := &Token{
		UserID:  userID,
		Expires: time.Now().Add(ttl),
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.PlainText = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.PlainText))
	token.Hash = hash[:]

	return token, nil
}

// AuthenticateToken validates a token from an HTTP request’s Authorization header.
func (t *Token) AuthenticateToken(r *http.Request) (*User, error) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("no authorization header received")
	}

	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("no valid authorization header received")
	}

	token := headerParts[1]
	if len(token) != 26 {
		return nil, errors.New("invalid token length")
	}

	tok, err := t.GetByToken(token)
	if err != nil {
		return nil, errors.New("no matching token found")
	}

	if tok.Expires.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	user, err := t.GetUserForToken(token)
	if err != nil {
		return nil, errors.New("no matching user found")
	}

	return user, nil
}

// ValidToken checks if a plaintext token is valid and unexpired.
func (t *Token) ValidToken(token string) (bool, error) {
	user, err := t.GetUserForToken(token)
	if err != nil {
		return false, errors.New("no matching user found")
	}

	if user.Token.PlainText == "" {
		return false, errors.New("no matching token found")
	}

	if user.Token.Expires.Before(time.Now()) {
		return false, errors.New("token expired")
	}

	return true, nil
}
