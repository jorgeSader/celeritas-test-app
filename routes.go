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

	a.App.Routes.Get("/create-user", func(w http.ResponseWriter, r *http.Request) {
		u := data.User{
			FirstName: "Jorge",
			LastName:  "Sader",
			Email:     "test@test.com",
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

		err = a.Models.Users.Update(*user)
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}

		fmt.Fprintf(w, "user with id %d has been updated to %s %s", id, user.FirstName, user.LastName)
	})

	//static routes
	fileServer := http.FileServer(http.Dir("./public/"))
	a.App.Routes.Handle("/public/*", http.StripPrefix("/public", fileServer))

	return a.App.Routes
}
