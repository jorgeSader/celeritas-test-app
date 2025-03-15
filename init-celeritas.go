package main

import (
	"github.com/jorgeSader/celeritas-test-app/middleware"
	"log"
	"os"

	"github.com/jorgeSader/celeritas-test-app/data"
	"github.com/jorgeSader/celeritas-test-app/handlers"

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

	myMiddleware := &middleware.Middleware{
		App: cel,
	}

	myHandlers := &handlers.Handlers{
		App: cel,
	}

	app := &application{
		App:        cel,
		Handlers:   myHandlers,
		Middleware: myMiddleware,
	}

	app.App.Routes = app.routes()

	app.Models = data.New(app.App.DB.Pool)

	myHandlers.Models = app.Models

	app.Middleware.Models = app.Models

	return app
}
