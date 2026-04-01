package handlers

import (
	"fmt"
	"io"
	database "mulit-minIO/internal/db"
	"mulit-minIO/internal/minio"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DownloadHandler struct {
	minioClient *minio.Client
	db          *database.Database
}

func NewDownloadHandler(minioClient *minio.Client, db *database.Database) *DownloadHandler {
	return &DownloadHandler{
		minioClient: minioClient,
		db:          db,
	}
}

func (h *DownloadHandler) DownloadFileByID(c *gin.Context) {
	fileID := c.Param("id")

	fileRecord, err := h.db.GetFileRecord(fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	reader, err := h.minioClient.DownloadFile(c.Request.Context(), fileRecord.ObjectName)
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

func (h *DownloadHandler) ListFiles(c *gin.Context) {
	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	files, err := h.db.ListFileRecords(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files":  files,
		"total":  len(files),
		"limit":  limit,
		"offset": offset,
	})
}

func (h *DownloadHandler) DeleteFile(c *gin.Context) {
	objectName := c.Param("object_name")

	if err := h.minioClient.DeleteFile(c.Request.Context(), objectName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.DeleteFileRecord(objectName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}