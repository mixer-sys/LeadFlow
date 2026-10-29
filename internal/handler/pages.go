package handler

import (
	"html/template"
	"net/http"
)

type PagesHandler struct {
	loginTmpl    *template.Template
	registerTmpl *template.Template
}

func NewPagesHandler() (*PagesHandler, error) {
	loginTmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		return nil, err
	}

	registerTmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		return nil, err
	}

	return &PagesHandler{
		loginTmpl:    loginTmpl,
		registerTmpl: registerTmpl,
	}, nil
}

func (h *PagesHandler) LoginPage(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.loginTmpl.Execute(w, nil); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *PagesHandler) RegisterPage(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.registerTmpl.Execute(w, nil); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
