package middleware

import (
	"github.com/jorgeSader/celeritas"
	"github.com/jorgeSader/celeritas-test-app/data"
)

type Middleware struct {
	App    *celeritas.Celeritas
	Models data.Models
}
