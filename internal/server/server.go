package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"systemdesigngame/internal/auth"
	"systemdesigngame/internal/level"
	"time"

	"github.com/joho/godotenv"
)

func InitApp() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("failed to load .env")
	}

	auth.JwtSecret = []byte(os.Getenv("JWT_SECRET"))

	levels, err := level.LoadLevels("data/levels.json")
	if err != nil {
		log.Fatal("failed to load levels: " + err.Error())
	}
	level.InitRegistry(levels)
}

func GracefulShutdown(srv *http.Server) {
	// setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// start http server in a separate goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Server failed: " + err.Error())
		}
	}()

	// wait for shutdown signal
	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv.Shutdown(shutdownCtx)
}
