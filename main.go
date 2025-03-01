package main

import (
	"github.com/jorgeSader/celeritas"
	"github.com/jorgeSader/celeritas-test-app/handlers"
)

type application struct {
	App      *celeritas.Celeritas
	Handlers *handlers.Handlers
}

func main() {
	c := InitApplication()
	c.App.ListenAndServe()
}
