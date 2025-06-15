package main

import (
	"github.com/joho/godotenv"
	"html/template"
	"net/http"
	"os"
	"systemdesigngame/internal/auth"
	"systemdesigngame/internal/server"
	"systemdesigngame/router"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("failed to load .env")
	}

	// set up JWT secret used for authentication
	auth.JwtSecret = []byte(os.Getenv("JWT_SECRET"))
	tmpl := template.Must(template.ParseGlob("static/*.html"))

	server.InitApp()

	mux := app.SetupRoutes(tmpl)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.GracefulShutdown(srv)
}
