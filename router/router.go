package app

import (
	"html/template"
	"net/http"
	"systemdesigngame/internal/auth"
	"systemdesigngame/router/handlers"
)

func SetupRoutes(tmpl *template.Template) *http.ServeMux {
	// initialize http routes and handlers
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/", &handlers.HomeHandler{Tmpl: tmpl})
	mux.Handle("/mode", auth.RequireAuth(&handlers.PlayHandler{Tmpl: tmpl}))

	mux.Handle("/play/", auth.RequireAuth(&handlers.PlayHandler{Tmpl: tmpl}))
	mux.Handle("/simulate", auth.RequireAuth(&handlers.SimulationHandler{}))
	mux.HandleFunc("/login", auth.LoginHandler)
	mux.HandleFunc("/callback", auth.CallbackHandler)

	return mux
}
