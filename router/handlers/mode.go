package handlers

import (
	"html/template"
	"net/http"
)

type ModeHandler struct {
	Tmpl *template.Template
}

func (m *ModeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data := struct {
		Title string
	}{
		Title: "Title",
	}

	if err := m.Tmpl.ExecuteTemplate(w, "game-mode.html", data); err != nil {
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}
