package handlers

import (
	"fmt"
	database "mulit-minIO/internal/db"
	"mulit-minIO/internal/minio"
	"mulit-minIO/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	minioClient *minio.Client
	db          *database.Database
}

func NewUploadHandler(minioClient *minio.Client, db *database.Database) *UploadHandler {
	return &UploadHandler{
		minioClient: minioClient,
		db:          db,
	}
}


func (h *UploadHandler) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	objectName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), header.Filename)

	result, err := h.minioClient.UploadFile(c.Request.Context(), objectName, file, 
		header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fileRecord := &database.FileRecord{
		ObjectName:   objectName,
		OriginalName: header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		Size:         result.Size,
		ETag:         result.ETag,
		UploadedBy:   c.GetString("user_id"),
	}

	if err := h.db.CreateFileRecord(fileRecord); err != nil {
		h.minioClient.DeleteFile(c.Request.Context(), objectName)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save metadata"})
		return
	}

	c.JSON(http.StatusOK, models.UploadResponse{
		File: models.File{
			ID:           fileRecord.ID,
			ObjectName:   fileRecord.ObjectName,
			OriginalName: fileRecord.OriginalName,
			ContentType:  fileRecord.ContentType,
			Size:         fileRecord.Size,
			ETag:         fileRecord.ETag,
			UploadedBy:   fileRecord.UploadedBy,
			CreatedAt:    fileRecord.CreatedAt,
			UpdatedAt:    fileRecord.UpdatedAt,
		},
	})
}

func (h *UploadHandler) MultipartUpload(c *gin.Context) {
	var req models.MultipartUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	partSize := req.PartSize
	if partSize == 0 {
		partSize = 5 * 1024 * 1024
	}

	totalParts := int((req.TotalSize + partSize - 1) / partSize)

	objectName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), req.FileName)


	presignedURLs := make([]string, totalParts)
	for i := 0; i < totalParts; i++ {
		url, err := h.minioClient.GeneratePresignedPutURL(c.Request.Context(), 
			objectName, 15*time.Minute)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		presignedURLs[i] = url
	}

	c.JSON(http.StatusOK, models.MultipartUploadResponse{
		UploadID:      objectName,
		PartSize:      partSize,
		TotalParts:    totalParts,
		PresignedURLs: presignedURLs,
	})
}

func (h *UploadHandler) CompleteMultipartUpload(c *gin.Context) {
	objectName := c.Param("object_name")
	
	fileInfo, err := h.minioClient.GetFileInfo(c.Request.Context(), objectName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	fileRecord := &database.FileRecord{
		ObjectName:   objectName,
		OriginalName: objectName,
		ContentType:  fileInfo.ContentType,
		Size:         fileInfo.Size,
		ETag:         fileInfo.ETag,
		UploadedBy:   c.GetString("user_id"),
	}

	if err := h.db.CreateFileRecord(fileRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save metadata"})
		return
	}

	c.JSON(http.StatusOK, models.File{
		ID:           fileRecord.ID,
		ObjectName:   fileRecord.ObjectName,
		OriginalName: fileRecord.OriginalName,
		ContentType:  fileRecord.ContentType,
		Size:         fileRecord.Size,
		ETag:         fileRecord.ETag,
		UploadedBy:   fileRecord.UploadedBy,
		CreatedAt:    fileRecord.CreatedAt,
		UpdatedAt:    fileRecord.UpdatedAt,
	})
}