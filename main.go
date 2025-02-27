package main

import "github.com/jorgeSader/celeritas"

type application struct {
	App *celeritas.Celeritas
}

func main() {
	c := InitApplication()
	c.App.ListenAndServe()
}
