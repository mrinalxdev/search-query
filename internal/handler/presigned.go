package handlers

import (
	"fmt"
	"io"
	database "mulit-minIO/internal/db"
	"mulit-minIO/internal/minio"
	"mulit-minIO/internal/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PresignedHandler struct {
	minioClient *minio.Client
	db          *database.Database
}

func NewPresignedHandler(minioClient *minio.Client, db *database.Database) *PresignedHandler {
	return &PresignedHandler{
		minioClient: minioClient,
		db:          db,
	}
}


func (h *PresignedHandler) GeneratePresignedURL(c *gin.Context) {
	var req models.PresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.GetFileRecord(req.ObjectName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	expiry := time.Duration(req.Expiry) * time.Minute
	if expiry == 0 {
		expiry = 15 * time.Minute
	}

	url, err := h.minioClient.GeneratePresignedURL(c.Request.Context(), req.ObjectName,
		minio.PresignedURLOptions{
			Expiry:    expiry,
			RequestMethod: req.Method,
		})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.PresignedURLResponse{
		URL:    url,
		Expiry: time.Now().Add(expiry),
	})
}

func (h *PresignedHandler) GeneratePresignedUploadURL(c *gin.Context) {
	objectName := c.Param("object_name")
	
	expiryStr := c.Query("expiry")
	expiry := 15 * time.Minute
	if expiryStr != "" {
		if minutes, err := time.ParseDuration(expiryStr + "m"); err == nil {
			expiry = minutes
		}
	}

	url, err := h.minioClient.GeneratePresignedPutURL(c.Request.Context(), objectName, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.PresignedURLResponse{
		URL:    url,
		Expiry: time.Now().Add(expiry),
	})
}


func (h *PresignedHandler) DownloadFile(c *gin.Context) {
	objectName := c.Param("object_name")
	fileRecord, err := h.db.GetFileRecord(objectName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	
	reader, err := h.minioClient.DownloadFile(c.Request.Context(), objectName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileRecord.OriginalName))
	c.Header("Content-Type", fileRecord.ContentType)
	c.Header("Content-Length", strconv.FormatInt(fileRecord.Size, 10))

	c.Stream(func(w io.Writer) bool {
		_, err := io.Copy(w, reader)
		return err == nil
	})
}