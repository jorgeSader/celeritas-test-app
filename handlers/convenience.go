package handlers

import (
	"fmt"
	"net/http"
)

// renderGo renders a Go template page.
func (h *Handlers) renderGo(w http.ResponseWriter, r *http.Request, tmpl string, args ...interface{}) error {
	var data interface{}
	if len(args) > 0 {
		data = args[0] // First arg is data, if provided
	}
	return h.App.Render.GoPage(w, r, tmpl, data)
}

// renderJet renders a Jet template page.
// args[0] is data, args[1] is variables; both default to nil if omitted.
func (h *Handlers) renderJet(w http.ResponseWriter, r *http.Request, tmpl string, args ...interface{}) error {
	var data, variables interface{}
	if len(args) > 0 {
		data = args[0] // First arg is data
	}
	if len(args) > 1 {
		variables = args[1] // Second arg is variables
	}
	return h.App.Render.JetPage(w, r, tmpl, variables, data)
}

// render is a convenience function that renders a page using the configured renderer.
// args[0] is data, args[1] is variables; both default to nil if omitted.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, tmpl string, args ...interface{}) error {
	var data, variables interface{}
	if len(args) > 0 {
		data = args[0] // First arg is data
	}
	if len(args) > 1 {
		variables = args[1] // Second arg is variables
	}
	switch h.App.Render.Renderer {
	case "go":
		return h.App.Render.GoPage(w, r, tmpl, data)
	case "jet":
		return h.App.Render.JetPage(w, r, tmpl, data, variables)
	default:
		return fmt.Errorf("unknown renderer type: %s", h.App.Render.Renderer)
	}
}
