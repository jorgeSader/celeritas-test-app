package data

import (
	"errors"
	"time"

	up "github.com/upper/db/v4"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the application, including their authentication token.
type User struct {
	ID        int       `db:"id,omitempty"` // Unique identifier, auto-incremented if omitted
	FirstName string    `db:"first_name"`   // User's first name
	LastName  string    `db:"last_name"`    // User's last name
	Email     string    `db:"email"`        // User's email address
	Active    string    `db:"active"`       // User's active status (e.g., "yes" or "no")
	Password  string    `db:"password"`     // Hashed password
	CreatedAt time.Time `db:"created_at"`   // Timestamp of user creation
	UpdatedAt time.Time `db:"updated_at"`   // Timestamp of last update
	Token     Token     `db:"-"`            // Associated token, not stored in the database
}

// Table returns the database table name for the User struct.
func (u *User) Table() string {
	// if legacy table had a different name we could change it here
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

// GetByEmail retrieves a user by their email address, including their latest token if available.
func (u *User) GetByEmail(email string) (*User, error) {
	var user User
	collection := upper.Collection(u.Table())
	res := collection.Find(up.Cond{"email =": email})
	err := res.One(&user)
	if err != nil {
		return nil, err
	}

	var token Token
	collection = upper.Collection(token.Table())
	res = collection.Find(up.Cond{"user_id =": user.ID, "expiry <": time.Now()}).OrderBy("created_at desc")
	err = res.One(&token)
	if err != nil {
		if err != up.ErrNilRecord && err != up.ErrNoMoreRows {
			return nil, err
		}
	}

	user.Token = token

	return &user, nil
}

// Get retrieves a user by their ID, including their latest token if available.
func (u *User) Get(id int) (*User, error) {
	var user User
	collection := upper.Collection(u.Table())
	res := collection.Find(up.Cond{"id =": id})
	err := res.One(&user)
	if err != nil {
		return nil, err
	}

	var token Token
	collection = upper.Collection(token.Table())
	res = collection.Find(up.Cond{"user_id =": user.ID, "expiry <": time.Now()}).OrderBy("created_at desc")
	err = res.One(&token)
	if err != nil {
		if err != up.ErrNilRecord && err != up.ErrNoMoreRows {
			return nil, err
		}
	}

	user.Token = token

	return &user, nil
}

// Update persists changes to an existing user in the database, setting the UpdatedAt timestamp.
func (u *User) Update(user User) error {
	user.UpdatedAt = time.Now()
	collection := upper.Collection(u.Table())
	res := collection.Find(up.Cond{"id =": user.ID})
	err := res.Update(&user)
	if err != nil {
		return err
	}

	return nil
}

// Delete removes a user from the database by their ID.
func (u *User) Delete(id int) error {
	collection := upper.Collection(u.Table())
	res := collection.Find(up.Cond{"id =": id})
	err := res.Delete()
	if err != nil {
		return err
	}
	return nil
}

// Insert adds a new user to the database, hashing their password and setting timestamps.
// It returns the inserted user's ID or an error if the operation fails.
// TODO: Modify to return up.ID type instead of int.
func (u *User) Insert(user User) (int, error) {
	newHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
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

	return id, nil
}

// ResetPassword updates the password for the user with the given ID, hashing the new password.
func (u *User) ResetPassword(id int, newPassword string) error {
	user, err := u.Get(id)
	if err != nil {
		return err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
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

// PasswordMatches verifies if the provided plaintext password matches the user's hashed password.
// It returns true if the passwords match, false if they don’t (or on error), and an error if the comparison fails unexpectedly.
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
