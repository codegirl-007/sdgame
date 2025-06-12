package main

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"systemdesigngame/internals/level"
	"time"
)

var tmpl = template.Must(template.ParseGlob("static/*.html"))

func main() {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", index)
	mux.HandleFunc("/game", game)
	mux.HandleFunc("/play/", play)
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	levels, err := level.LoadLevels("data/levels.json")
	if err != nil {
		panic("failed to load levels: " + err.Error())
	}
	level.InitRegistry(levels)

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

func play(w http.ResponseWriter, r *http.Request) {
	levelName := r.URL.Path[len("/play/"):]
	levelName, err := url.PathUnescape(levelName)
	if err != nil {
		http.Error(w, "Invalid level name", http.StatusBadRequest)
		return
	}

	lvl, err := level.GetLevel(strings.ToLower(levelName), level.DifficultyEasy)
	if err != nil {
		http.Error(w, "Level not found"+err.Error(), http.StatusNotFound)
		return
	}
	allLevels := level.AllLevels()
	data := struct {
		Levels []level.Level
		Level  *level.Level
	}{
		Levels: allLevels,
		Level:  lvl,
	}

	tmpl.ExecuteTemplate(w, "game.html", data)
}
