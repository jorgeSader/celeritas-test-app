package main

import (
	"github.com/jorgeSader/celeritas"
	"github.com/jorgeSader/celeritas-test-app/data"
	"github.com/jorgeSader/celeritas-test-app/handlers"
	"github.com/jorgeSader/celeritas-test-app/middleware"
)

type application struct {
	App        *celeritas.Celeritas
	Handlers   *handlers.Handlers
	Models     data.Models
	Middleware *middleware.Middleware
}

func main() {
	c := InitApplication()
	c.App.ListenAndServe()
}
