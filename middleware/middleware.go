package middleware

import (
	"github.com/jorgeSader/devify"
	"github.com/jorgeSader/devify-test-app/data"
)

type Middleware struct {
	App    *devify.Devify
	Models data.Models
}
