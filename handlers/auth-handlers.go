package handlers

import (
	"net/http"

	up "github.com/upper/db/v4"
)

func (h *Handlers) UserLogin(w http.ResponseWriter, r *http.Request) {
	err := h.App.Render.Page(w, r, "login", nil, nil)
	if err != nil {
		h.App.ErrorLog.Println(err)
	}
}

func (h *Handlers) PostUserLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.Write([]byte(err.Error()))
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	user, err := h.Models.Users.GetByEmail(email)
	if err != nil {
		if err == up.ErrNilRecord || err == up.ErrNoMoreRows {
			w.Write([]byte("No user with that email was found!"))
			return
		}
		w.Write([]byte(err.Error()))
	}

	passwordMatches, err := user.PasswordMatches(password)
	if err != nil {
		w.Write([]byte("Error validating password."))
	}
	if !passwordMatches {
		w.Write([]byte("Invalid password!"))
	}

	h.App.Session.Put(r.Context(), "userID", user.ID)

	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	h.App.Session.RenewToken(r.Context())
	h.App.Session.Remove(r.Context(), "userID")
	http.Redirect(w, r, "/users/login", http.StatusSeeOther)
}
