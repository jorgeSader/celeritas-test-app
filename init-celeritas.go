package main

import (
	"log"
	"myappgithub.com/jorgeSader/celeritas-test-app/handlers"
	"os"

	"github.com/jorgeSader/celeritas"
)

func InitApplication() *application {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// init celeritas
	cel := &celeritas.Celeritas{}
	err = cel.New(path)
	if err != nil {
		log.Fatal(err)
	}

	cel.AppName = "myapp"

	myHandlers := &handlers.Handlers{
		App: cel,
	}

	app := &application{
		App:      cel,
		Handlers: myHandlers,
	}

	app.App.Routes = app.routes()

	return app
}