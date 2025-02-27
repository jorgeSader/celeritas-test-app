package main

import (
	"log"
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

	cel.InfoLog.Println("Debug is set to", cel.Debug)

	app := &application{
		App: cel,
	}

	return app
}
