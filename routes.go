package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jorgeSader/celeritas-test-app/data"
)

func (a *application) routes() *chi.Mux {

	r := chi.NewRouter()

	//middleware must come before routes
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// add routes here
	a.App.Routes.Get("/", a.Handlers.Home)
	a.App.Routes.Get("/go-page", a.Handlers.GoPage)
	a.App.Routes.Get("/jet-page", a.Handlers.JetPage)
	a.App.Routes.Get("/sessions", a.Handlers.SessionTest)

	a.App.Routes.Get("/users/login", a.Handlers.UserLogin)
	a.App.Routes.Post("/users/login", a.Handlers.PostUserLogin)
	a.App.Routes.Get("/users/logout", a.Handlers.Logout)

	a.App.Routes.Get("/form", a.Handlers.Form)
	a.App.Routes.Post("/form", a.Handlers.PostForm)

	a.App.Routes.Get("/create-user", func(w http.ResponseWriter, r *http.Request) {
		u := data.User{
			FirstName: "Jorge",
			LastName:  "Sader",
			Email:     "test@email.com",
			Active:    1,
			Password:  "Test@123",
		}

		id, err := a.Models.Users.Insert(u)
		if err != nil {
			a.App.ErrorLog.Println(err)
		}
		fmt.Fprintf(w, "%d: %s", id, u.FirstName)
	})

	a.App.Routes.Get("/get-all-users", func(w http.ResponseWriter, r *http.Request) {
		users, err := a.Models.Users.GetAll()
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}
		for _, user := range users {
			fmt.Fprintln(w, user.LastName)
		}

	})

	a.App.Routes.Get("/get-user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))

		user, err := a.Models.Users.Get(id)
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}
		fmt.Fprintf(w, "id %d belongs to user %s %s", id, user.FirstName, user.LastName)

	})

	a.App.Routes.Get("/update-user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))

		user, err := a.Models.Users.Get(id)
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}

		user.LastName = a.App.RandomString(11)

		///////////////////////////////////
		// Testing new validator package //
		///////////////////////////////////
		validator := a.App.Validator(r) // r.Form ignored since we overwrite Data

		user.Validate(validator)

		//// Mock form data with some intentional failures
		//validator.Data = url.Values{
		//	"first_name": []string{"Jo"},          // Too short for Between 3, 50
		//	"last_name":  []string{user.LastName}, // 11 chars, should pass 5-15
		//	"email":      []string{"joe@shmoe"},   // Invalid email
		//	"active":     []string{"1"},           // Invalid int
		//	"created_at": []string{"01-13-2025"},  // Invalid date (month 13)
		//	"username":   []string{"user name"},   // Has spaces
		//}

		//// Chain validation rules to test various scenarios
		//validator.Required("first_name", "last_name", "email", "active", "created_at", "username").
		//	Between("first_name", 3, 50, "First name must be 3-50 characters").
		//	MinLength("last_name", 5, "Last name must be at least 5 characters").
		//	MaxLength("last_name", 15, "Last name must be no more than 15 characters").
		//	IsEmail("email", "Must be a valid email address").
		//	IsInt("active", "Active must be an integer").
		//	IsBoolean("active", "Active must be an boolean").
		//	IsDate("created_at", "Invalid date format").
		//	Contains("username", "@", "Username must contain @").
		//	HasNoSpaces("username", "Username must not contain spaces")

		// Output results
		if !validator.Valid() {
			w.Write([]byte("Validation failed:\n"))
			for field, err := range validator.Errors {
				fmt.Fprintf(w, "%s: %s\n", field, err)
			}
			return
		}

		// If we reach here, update the user (though this is just a test)
		err = a.Models.Users.Update(*user)
		if err != nil {
			a.App.ErrorLog.Println("Update failed:", err)
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "User with id %d updated to %s %s (all validations passed!)", id, user.FirstName, user.LastName)
	})

	//static routes
	fileServer := http.FileServer(http.Dir("./public/"))
	a.App.Routes.Handle("/public/*", http.StripPrefix("/public", fileServer))

	return a.App.Routes
}
