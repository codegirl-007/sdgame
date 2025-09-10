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

	mux.Handle("/play/{levelId}", auth.RequireAuth(&handlers.PlayHandler{Tmpl: tmpl}))
	mux.Handle("/simulate", auth.RequireAuth(&handlers.SimulationHandler{}))
	mux.Handle("/success", auth.RequireAuth(&handlers.SuccessHandler{Tmpl: tmpl}))
	mux.Handle("/failure", auth.RequireAuth(&handlers.FailureHandler{Tmpl: tmpl}))
	mux.HandleFunc("/login", auth.LoginHandler)
	mux.HandleFunc("/callback", auth.CallbackHandler)
	mux.HandleFunc("/ws", handlers.Messages)

	return mux
}
