package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CloudyKit/jet/v6"
	"github.com/jorgeSader/devify"
	"github.com/jorgeSader/devify-test-app/data"
)

type Handlers struct {
	App    *devify.Devify
	Models data.Models
}

func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	err := h.render(w, r, "home")
	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) GoPage(w http.ResponseWriter, r *http.Request) {
	err := h.renderGo(w, r, "go-template")
	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) JetPage(w http.ResponseWriter, r *http.Request) {
	err := h.renderJet(w, r, "jet-template")
	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) SessionTest(w http.ResponseWriter, r *http.Request) {
	myData := "bar"

	h.App.Session.Put(r.Context(), "foo", myData)
	myValue := h.App.Session.GetString(r.Context(), "foo")

	vars := make(jet.VarMap)
	vars.Set("foo", myValue)

	log.Printf("Template data: %+v", vars)

	err := h.render(w, r, "sessions", nil, vars)
	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) JSON(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID      int      `json:"id"`
		Name    string   `json:"name"`
		Hobbies []string `json:"hobbies"`
	}

	payload.ID = 10
	payload.Name = "john doe"
	payload.Hobbies = []string{"aikido", "rugby", "guitar"}

	err := h.App.WriteJSON(w, http.StatusOK, &payload)
	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) XML(w http.ResponseWriter, r *http.Request) {
	type Payload struct {
		ID      int      `xml:"id"`
		Name    string   `xml:"name"`
		Hobbies []string `xml:"hobbies>hobby"`
	}
	var payload Payload
	payload.ID = 10
	payload.Name = "john doe"
	payload.Hobbies = []string{"aikido", "rugby", "guitar"}

	err := h.App.WriteXML(w, http.StatusOK, &payload)
	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) DownloadFile(w http.ResponseWriter, r *http.Request) {
	h.App.DownloadFile(w, r, "./public/images", "devify-sq-colorFont.png")
}

func (h *Handlers) TestCrypto(w http.ResponseWriter, r *http.Request) {
	plainText := "Hello World!"
	fmt.Fprintln(w, "unencrypted data: "+plainText)
	encrypted, err := h.encrypt(plainText)
	if err != nil {
		h.App.ErrorLog.Println("error encrypting data:", err)
		h.App.Error500(w)
		return
	}

	fmt.Fprintln(w, "encrypted data: "+encrypted)

	decrypted, err := h.decrypt(encrypted)
	if err != nil {
		h.App.ErrorLog.Println("error decrypting data:", err)
		h.App.Error500(w)
		return
	}

	fmt.Fprintln(w, "decrypted data: "+decrypted)
}
