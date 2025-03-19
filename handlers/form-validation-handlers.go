package handlers

import (
	"fmt"
	"net/http"

	"github.com/CloudyKit/jet/v6"
	"github.com/jorgeSader/devify-test-app/data"
)

func (h *Handlers) Form(w http.ResponseWriter, r *http.Request) {
	vars := make(jet.VarMap)
	validator := h.App.Validator(r)
	vars.Set("validator", validator)
	vars.Set("user", data.User{})

	err := h.App.Render.Page(w, r, "form", nil, vars)
	if err != nil {
		h.App.ErrorLog.Println(err)
	}
}

func (h *Handlers) PostForm(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.App.ErrorLog.Println(err)
		return
	}

	validator := h.App.Validator(r)

	validator.Required("first_name", "last_name", "email").
		Between("first_name", 2, 5).
		Between("last_name", 2, 5).
		IsEmail("email")

	if !validator.Valid() {
		vars := jet.VarMap{}
		vars.Set("validator", validator)

		user := data.User{
			FirstName: r.FormValue("first_name"),
			LastName:  r.FormValue("last_name"),
			Email:     r.FormValue("email"),
		}

		vars.Set("user", user)

		err := h.App.Render.Page(w, r, "form", nil, vars)
		if err != nil {
			h.App.ErrorLog.Println("error rendering:", err)
			return
		}
		return
	}
	fmt.Fprint(w, "Valid data")
}
