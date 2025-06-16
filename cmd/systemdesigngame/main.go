package main

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"systemdesigngame/internal/auth"
	"systemdesigngame/internal/server"
	"systemdesigngame/router"

	"github.com/joho/godotenv"
)

func main() {
	devMode := flag.Bool("dev", false, "load .env (local dev)")
	flag.Parse()

	fmt.Printf("devmode: %v", *devMode)
	if *devMode {
		if err := godotenv.Load(); err != nil {
			panic("failed to load .env")
		}

	}

	// set up JWT secret used for authentication
	auth.JwtSecret = []byte(os.Getenv("JWT_SECRET"))

	if len(auth.JwtSecret) == 0 {
		panic("JWT_SECRET is not set")
	}

	tmpl := template.Must(template.ParseGlob("static/*.html"))

	server.InitApp()

	mux := app.SetupRoutes(tmpl)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.GracefulShutdown(srv)
}
