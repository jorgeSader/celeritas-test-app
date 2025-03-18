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
	a.get("/", a.Handlers.Home)
	a.get("/go-page", a.Handlers.GoPage)
	a.get("/jet-page", a.Handlers.JetPage)
	a.get("/sessions", a.Handlers.SessionTest)

	a.get("/users/login", a.Handlers.UserLogin)
	a.post("/users/login", a.Handlers.PostUserLogin)
	a.get("/users/logout", a.Handlers.Logout)

	a.get("/form", a.Handlers.Form)
	a.post("/form", a.Handlers.PostForm)

	a.get("/json", a.Handlers.JSON)
	a.get("/xml", a.Handlers.XML)
	a.get("/download-file", a.Handlers.DownloadFile)

	a.get("/crypto", a.Handlers.TestCrypto)

	a.get("/create-user", func(w http.ResponseWriter, r *http.Request) {
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

	a.get("/get-all-users", func(w http.ResponseWriter, r *http.Request) {
		users, err := a.Models.Users.GetAll()
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}
		for _, user := range users {
			fmt.Fprintln(w, user.LastName)
		}

	})

	a.get("/get-user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))

		user, err := a.Models.Users.Get(id)
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}
		fmt.Fprintf(w, "id %d belongs to user %s %s", id, user.FirstName, user.LastName)

	})

	a.get("/update-user/{id}", func(w http.ResponseWriter, r *http.Request) {
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
