package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

var Version = "dev"

func main() {
	// Initialize Gin
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"version": Version,
		})
	})

	// Version endpoint
	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": Version,
			"service": "housepoints-go",
		})
	})

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "HousePoints Go API",
			"version": Version,
			"docs":    "/api/docs",
		})
	})

	// Server configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("✅ Server exited")
}
