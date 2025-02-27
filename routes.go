package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
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

	//static routes
	fileServer := http.FileServer(http.Dir("./public/"))
	a.App.Routes.Handle("/public/*", http.StripPrefix("/public", fileServer))

	return a.App.Routes
}