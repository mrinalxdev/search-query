package main

import (
	"context"
	"fmt"
	"log"
	"mulit-minIO/internal/config"
	database "mulit-minIO/internal/db"
	handlers "mulit-minIO/internal/handler"
	"mulit-minIO/internal/minio"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting application with config: %s", cfg)
	minioClient, err := minio.NewClient(&cfg.MinIO)
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	ctx := context.Background()
	if err := minioClient.EnsureBucket(ctx); err != nil {
		log.Fatalf("Failed to ensure bucket: %v", err)
	}
	log.Printf("Bucket '%s' is ready", cfg.MinIO.Bucket)
	db, err := database.NewDatabase(&cfg.DB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")
	uploadHandler := handlers.NewUploadHandler(minioClient, db)
	presignedHandler := handlers.NewPresignedHandler(minioClient, db)
	downloadHandler := handlers.NewDownloadHandler(minioClient, db)
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "ETag"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	r.POST("/api/v1/upload", uploadHandler.UploadFile)
	r.POST("/api/v1/upload/multipart", uploadHandler.MultipartUpload)
	r.POST("/api/v1/upload/complete/:object_name", uploadHandler.CompleteMultipartUpload)
	r.POST("/api/v1/presigned", presignedHandler.GeneratePresignedURL)
	r.GET("/api/v1/presigned/upload/:object_name", presignedHandler.GeneratePresignedUploadURL)
	r.GET("/api/v1/download/:object_name", presignedHandler.DownloadFile)
	r.GET("/api/v1/files/:id", downloadHandler.DownloadFileByID)
	r.GET("/api/v1/files", downloadHandler.ListFiles)
	r.DELETE("/api/v1/files/:object_name", downloadHandler.DeleteFile)

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %d", cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}