package handlers

import (
	"html/template"
	"net/http"
)

type HomeHandler struct {
	Tmpl *template.Template
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := struct{ Title string }{Title: "Title"}
	if err := h.Tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}
