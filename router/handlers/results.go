package handlers

import (
	"html/template"
	"net/http"
)

type ResultHandler struct {
	Tmpl *template.Template
}

func (r *ResultHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data := struct {
		Title string
	}{
		Title: "Title",
	}

	if err := r.Tmpl.ExecuteTemplate(w, "success.html", data); err != nil {
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}
