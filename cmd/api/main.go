package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/Ayan25844/subscout/docs"
	"github.com/Ayan25844/subscout/internal/database"
	"github.com/Ayan25844/subscout/internal/handlers"
	"github.com/joho/godotenv"
)

// @title           SubScout Management API
// @version         1.0
// @description     A high-performance subscription tracking service built with Go and PostgreSQL.
// @description     This API handles user authentication, subscription lifecycle management, and expense analytics.
// @termsOfService  http://swagger.io/terms/

// @contact.name    Ayan Chatterjee
// @contact.url     https://github.com/Ayan25844
// @contact.email   ayan25844@gmail.com

// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1
// @query.collection.format multi

// @securityDefinitions.apikey  Bearer
// @in                          header
// @name                        Authorization
// @description                 JWT Authorization header using the Bearer scheme.
// @description                 Example: "Authorization: Bearer {token}"
func main() {
	godotenv.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dbService, err := database.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer dbService.Close()
	router := handlers.RegisterRoutes(dbService)
	srv := &http.Server{
		Addr:         ":" + os.Getenv("PORT"),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.Printf("SubScout Server started on port %s", os.Getenv("PORT"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	<-done
	log.Println("Server Stopping...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server Exited Gracefully")
}
