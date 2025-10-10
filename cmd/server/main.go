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

	"github.com/JunoAX/housepoints-go/internal/auth"
	"github.com/JunoAX/housepoints-go/internal/database"
	"github.com/JunoAX/housepoints-go/internal/handlers"
	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/gin-gonic/gin"
)

var Version = "dev"

func main() {
	ctx := context.Background()

	// Get database configuration from environment
	platformDBURL := os.Getenv("PLATFORM_DATABASE_URL")
	if platformDBURL == "" {
		// Default for local development / production
		platformDBURL = "postgres://postgres:HP_Sec2025_O0mZVY90R1Yg8L@10.1.10.20:5432/housepoints_platform?sslmode=disable"
	}

	// Initialize platform database
	log.Println("📦 Connecting to platform database...")
	platformDB, err := database.NewPlatformDB(ctx, platformDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to platform database: %v", err)
	}
	defer platformDB.Close()
	log.Println("✅ Platform database connected")

	// Initialize family database manager
	familyDBManager := database.NewFamilyDBManager(platformDB)
	defer familyDBManager.Close()
	log.Println("✅ Family database manager initialized")

	// Initialize JWT service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	jwtService := auth.NewJWTService(jwtSecret, "housepoints-go")
	log.Println("✅ JWT service initialized")

	// Initialize Gin
	r := gin.Default()

	// Apply family middleware globally
	baseDomain := os.Getenv("BASE_DOMAIN")
	if baseDomain == "" {
		baseDomain = "housepoints.ai"
	}
	r.Use(middleware.FamilyMiddleware(familyDBManager, baseDomain))

	// Health check (no family required)
	r.GET("/health", func(c *gin.Context) {
		// Check platform DB health
		dbHealthy := platformDB.Health(ctx) == nil

		c.JSON(200, gin.H{
			"status":      "healthy",
			"version":     Version,
			"db_platform": dbHealthy,
		})
	})

	// Detailed health check with pool stats
	r.GET("/health/detailed", func(c *gin.Context) {
		dbHealthy := platformDB.Health(ctx) == nil
		poolStats := familyDBManager.PoolStats()

		c.JSON(200, gin.H{
			"status":      "healthy",
			"version":     Version,
			"db_platform": dbHealthy,
			"db_pools":    poolStats,
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

	// Family info endpoint (requires family subdomain)
	r.GET("/api/family/info", middleware.RequireFamily(), func(c *gin.Context) {
		family, _ := middleware.GetFamily(c)
		familyDB, _ := middleware.GetFamilyDB(c)

		// Test query on family database
		var dbName string
		err := familyDB.QueryRow(ctx, "SELECT current_database()").Scan(&dbName)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to query family database"})
			return
		}

		c.JSON(200, gin.H{
			"family_id":   family.ID,
			"family_slug": family.Slug,
			"family_name": family.Name,
			"plan":        family.Plan,
			"status":      family.Status,
			"database":    dbName,
		})
	})

	// Authentication endpoints
	r.POST("/api/auth/login", middleware.RequireFamily(), handlers.Login(jwtService))
	r.POST("/api/auth/google/mobile", middleware.RequireFamily(), handlers.GoogleMobileAuth(jwtService))

	// Web OAuth endpoints (no RequireFamily - handles multi-tenant via state)
	r.GET("/api/auth/google/init", handlers.GoogleWebInit())
	r.GET("/api/auth/google/callback", handlers.GoogleWebCallback(jwtService))

	// Protected API routes (require authentication)
	protected := r.Group("/api")
	protected.Use(middleware.RequireFamily(), middleware.RequireAuth(jwtService))
	{
		// Users endpoints
		protected.GET("/users", handlers.ListUsers)
		protected.POST("/users", handlers.CreateUser)
		protected.GET("/users/me", handlers.GetCurrentUser)
		protected.PATCH("/users/me", handlers.UpdateCurrentUserProfile)
		protected.PUT("/users/me/preferences", handlers.UpdateCurrentUserPreferences)
		protected.GET("/users/:id", handlers.GetUser)
		protected.PUT("/users/:id", handlers.UpdateUser)
		protected.DELETE("/users/:id", handlers.DeleteUser)
		protected.GET("/users/:id/points", handlers.GetUserPoints)
		protected.GET("/users/:id/transactions", handlers.GetUserTransactions)
		protected.GET("/users/:id/stats", handlers.GetUserStats)
		protected.GET("/users/:id/redeemed-rewards", handlers.GetRedeemedRewards)

		// Chores endpoints
		protected.GET("/chores", handlers.ListChores)
		protected.POST("/chores", handlers.CreateChore)
		protected.GET("/chores/:id", handlers.GetChore)
		protected.PUT("/chores/:id", handlers.UpdateChore)
		protected.DELETE("/chores/:id", handlers.DeleteChore)

		// Assignments endpoints (read)
		protected.GET("/assignments", handlers.ListAssignments)
		protected.GET("/assignments/my-assignments", handlers.GetMyAssignments)
		protected.GET("/assignments/:id", handlers.GetAssignment)

		// Assignments endpoints (write)
		protected.POST("/assignments", handlers.CreateAssignment)
		protected.POST("/assignments/:id/claim", handlers.ClaimAssignment)
		protected.POST("/assignments/:id/complete", handlers.CompleteAssignment)
		protected.POST("/assignments/:id/verify", handlers.VerifyAssignment)

		// Rewards endpoints
		protected.GET("/rewards", handlers.ListRewards)
		protected.POST("/rewards", handlers.CreateReward)
		protected.PUT("/rewards/:id", handlers.UpdateReward)
		protected.DELETE("/rewards/:id", handlers.DeleteReward)
		protected.POST("/rewards/:id/redeem", handlers.RedeemReward)

		// Leaderboard endpoints
		protected.GET("/leaderboard/weekly", handlers.GetWeeklyLeaderboard)
		protected.GET("/leaderboard/alltime", handlers.GetAllTimeLeaderboard)

		// Family Schedule endpoints
		protected.GET("/schedule", handlers.GetFamilySchedule)

		// Settings endpoints
		protected.GET("/settings", handlers.GetSettings)
		protected.GET("/settings/:key", handlers.GetSetting)
		protected.PUT("/settings/:key", handlers.UpdateSetting)

		// Reports endpoints
		protected.GET("/reports/weekly-summary", handlers.GetWeeklySummary)
		protected.GET("/reports/child-performance/:child_id", handlers.GetChildPerformance)
		protected.GET("/reports/category-breakdown", handlers.GetCategoryBreakdown)
		protected.GET("/reports/performance-trends", handlers.GetPerformanceTrends)
	}

	// Demo-only endpoints (for testing without auth)
	r.GET("/api/demo/chores", middleware.RequireFamily(), middleware.DemoOnly(), handlers.ListChores)

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
		log.Printf("🌐 Base domain: %s", baseDomain)
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
