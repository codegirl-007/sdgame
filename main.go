package main

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"systemdesigngame/internals/level"
	"time"
)

var tmpl = template.Must(template.ParseGlob("static/*.html"))

func main() {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", index)
	mux.HandleFunc("/game", game)
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		srv.ListenAndServe()
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv.Shutdown(shutdownCtx)
}

func index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title string
	}{
		Title: "Title",
	}

	tmpl.ExecuteTemplate(w, "index.html", data)
}

func game(w http.ResponseWriter, r *http.Request) {
	var err error
	levels, err := level.LoadLevels("data/levels.json")
	if err != nil {
		panic("failed to load levels: " + err.Error())
	}

	data := struct {
		Levels []level.Level
	}{
		Levels: levels,
	}
	tmpl.ExecuteTemplate(w, "game.html", data)
}
