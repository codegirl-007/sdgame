package main

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var tmpl = template.Must(template.ParseFiles("static/index.html"))

func main() {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", index)
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

	tmpl.Execute(w, data)
}
