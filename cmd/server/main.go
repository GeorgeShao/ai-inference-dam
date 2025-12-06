package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/georgeshao/ai-inference-dam/internal/api"
	"github.com/georgeshao/ai-inference-dam/internal/dispatcher"
	"github.com/georgeshao/ai-inference-dam/internal/storage"
	"github.com/georgeshao/ai-inference-dam/internal/storage/sqlite"
)

const (
	DefaultPort        = ":8080"
	DefaultStoragePath = "./data/inference_dam.db"
)

func main() {
	port := getEnv("PORT", DefaultPort)
	if port[0] != ':' {
		port = ":" + port
	}
	storagePath := getEnv("STORAGE_PATH", DefaultStoragePath)

	// Initialize storage
	store, err := sqlite.New(storagePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	if err := ensureDefaultNamespace(store); err != nil {
		log.Fatalf("Failed to create default namespace: %v", err)
	}

	// Initialize dispatcher
	dispatcherConfig := dispatcher.DefaultConfig()
	d := dispatcher.New(store, dispatcherConfig)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		BodyLimit:    10 * 1024 * 1024, // 10MB
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Namespace, X-Provider-Endpoint, X-Provider-Key",
	}))

	// Setup routes
	api.SetupRoutes(app, store, d)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	// Start server
	log.Printf("Starting AI Inference Dam server on %s", port)
	if err := app.Listen(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func ensureDefaultNamespace(store storage.Store) error {
	ctx := context.Background()

	ns, err := store.GetNamespace(ctx, "default")
	if err != nil {
		return err
	}

	if ns == nil {
		now := time.Now()
		return store.CreateNamespace(ctx, &storage.NamespaceRecord{
			Name:        "default",
			Description: "Default namespace",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	return nil
}
