package main

import (
	"github.com/jorgeSader/devify"
	"github.com/jorgeSader/devify-test-app/data"
	"github.com/jorgeSader/devify-test-app/handlers"
	"github.com/jorgeSader/devify-test-app/middleware"
)

type application struct {
	App        *devify.Devify
	Handlers   *handlers.Handlers
	Models     data.Models
	Middleware *middleware.Middleware
}

func main() {
	c := InitApplication()
	c.App.ListenAndServe()
}
