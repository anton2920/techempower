package handler

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	"github.com/anton2920/techempower/clean/internal/service"
)

type Handler struct {
	fortunes  service.FortunesService

	templates *template.Template
	mux *http.ServeMux
}

func New(fortunesService service.FortunesService) (*Handler, error) {
	var h Handler
	var err error

	h.fortunes = fortunesService
	h.templates, err = template.ParseFiles("templates/fortunes.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse template files: %w", err)
	}

	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/fortunes", h.FortunesHandler)

	return &h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) FortunesHandler(w http.ResponseWriter, r *http.Request) {
	fortunes, err := h.fortunes.GetAllSorted(context.Background())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to get fortunes: %v", err)
		return
	}

	if err := h.templates.ExecuteTemplate(w, "fortunes.tmpl", fortunes); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to render fortunes template: %v", err)
		return
	}
}
