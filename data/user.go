package data

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/upper/db/v4"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user entity in the database.
type User struct {
	ID        int       `db:"id,omitempty"`
	FirstName string    `db:"first_name"`
	LastName  string    `db:"last_name"`
	Email     string    `db:"email"`
	Active    int       `db:"user_active"`
	Password  string    `db:"password"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Token     Token     `db:"-"`
}

// Table returns the database table name for the User model.
func (u *User) Table() string {
	return "users"
}

// GetAll retrieves all users from the database, ordered by last name.
func (u *User) GetAll() ([]*User, error) {
	collection := upper.Collection(u.Table())
	var all []*User
	res := collection.Find().OrderBy("last_name")
	err := res.All(&all)
	if err != nil {
		return nil, err
	}
	return all, nil
}

// GetByEmail retrieves a user by their email address.
// It includes the most recent non-expired token, if available.
func (u *User) GetByEmail(email string) (*User, error) {
	var user User
	collection := upper.Collection(u.Table())
	res := collection.Find(db.Cond{"email =": email})
	err := res.One(&user)
	if err != nil {
		return nil, err
	}

	var token Token
	collection = upper.Collection(token.Table())
	res = collection.Find(db.Cond{"user_id =": user.ID, "expiry >": time.Now()}).OrderBy("created_at desc")
	err = res.One(&token)
	if err != nil {
		if !errors.Is(err, db.ErrNilRecord) && !errors.Is(err, db.ErrNoMoreRows) {
			return nil, err
		}
	}

	user.Token = token
	return &user, nil
}

// Get retrieves a user by their ID.
// It includes the most recent non-expired token, if available.
func (u *User) Get(id int) (*User, error) {
	var user User
	collection := upper.Collection(u.Table())
	res := collection.Find(db.Cond{"id =": id})
	err := res.One(&user)
	if err != nil {
		return nil, err
	}

	var token Token
	collection = upper.Collection(token.Table())
	res = collection.Find(db.Cond{"user_id =": user.ID, "expiry >": time.Now()}).OrderBy("created_at desc")
	err = res.One(&token)
	if err != nil {
		if !errors.Is(err, db.ErrNilRecord) && !errors.Is(err, db.ErrNoMoreRows) {
			return nil, err
		}
	}

	user.Token = token
	return &user, nil
}

// Update modifies an existing user in the database.
// It updates the UpdatedAt timestamp to the current time.
func (u *User) Update(user User) error {
	user.UpdatedAt = time.Now()
	collection := upper.Collection(u.Table())
	res := collection.Find(db.Cond{"id =": user.ID})
	err := res.Update(&user)
	if err != nil {
		return err
	}
	return nil
}

// Delete removes a user from the database by their ID.
func (u *User) Delete(id int) error {
	collection := upper.Collection(u.Table())
	res := collection.Find(db.Cond{"id =": id})
	err := res.Delete()
	if err != nil {
		return err
	}
	return nil
}

// Insert adds a new user to the database.
// It hashes the password with bcrypt, sets timestamps, and returns the new user’s ID, updating the User struct.
func (u *User) Insert(user User) (int, error) {
	cost := 12
	if c := os.Getenv("BCRYPT_COST"); c != "" {
		if i, err := strconv.Atoi(c); err == nil {
			cost = i
		}
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), cost)
	if err != nil {
		return 0, err
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.Password = string(newHash)

	collection := upper.Collection(u.Table())
	res, err := collection.Insert(user)
	if err != nil {
		return 0, err
	}

	id := GetInsertID(res.ID())
	user.ID = id // Update the struct with the new ID
	return id, nil
}

// ResetPassword updates a user’s password by their ID.
// It hashes the new password with bcrypt and updates the user record.
func (u *User) ResetPassword(id int, newPassword string) error {
	user, err := u.Get(id)
	if err != nil {
		return err
	}

	cost := 12
	if c := os.Getenv("BCRYPT_COST"); c != "" {
		if i, err := strconv.Atoi(c); err == nil {
			cost = i
		}
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), cost)
	if err != nil {
		return err
	}

	user.Password = string(newHash)
	err = u.Update(*user)
	if err != nil {
		return err
	}
	return nil
}

// PasswordMatches verifies if the provided plaintext password matches the stored hash.
// It returns true if they match, false otherwise, with an error only on bcrypt failure.
func (u *User) PasswordMatches(plainText string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plainText))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}
